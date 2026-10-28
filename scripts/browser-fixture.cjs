/* Headless Chrome + a real loopback origin. No package install or cluster needed. */
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const http = require('node:http');
const { spawn } = require('node:child_process');
const { once } = require('node:events');

const candidates = [process.env.CHROME_BIN, '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/usr/bin/chromium', '/usr/bin/chromium-browser', '/usr/bin/google-chrome'];
const chrome = candidates.find(candidate => candidate && fs.existsSync(candidate));
const root = path.resolve(__dirname, '../go/static');
const pages = {'/': 'index.html', '/autodocs-dashboard-page': 'autodocs-dashboard.html', '/versions-dashboard-page': 'versions-dashboard.html'};

async function browserFixture(t, options = {}) {
	const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'autodocs-chrome-'));
	const requests = [];
	const server = http.createServer((request, response) => {
		const url = new URL(request.url, 'http://localhost');
		requests.push({path: url.pathname, method: request.method});
		if (url.pathname === '/empty') { response.end('<!doctype html><title>Empty</title>'); return; }
		let filename = pages[url.pathname];
		if (!filename && url.pathname.startsWith('/static/')) filename = path.basename(url.pathname);
		if (!filename || !fs.existsSync(path.join(root, filename))) { response.writeHead(404); response.end(); return; }
		let source = fs.readFileSync(path.join(root, filename));
		if (filename.endsWith('.html')) {
			const setup = typeof options.setup === 'function' ? options.setup(url.pathname) : options.setup;
			source = source.toString().replace('<script src="/static/dashboard-common.js"></script>', '<script>window.__fixtureLoad = crypto.randomUUID();\n' + (setup || '') + '</script><script src="/static/dashboard-common.js"></script>');
		}
		response.setHeader('Content-Type', filename.endsWith('.css') ? 'text/css' : filename.endsWith('.js') ? 'text/javascript' : 'text/html');
		response.end(source);
	});
	await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
	const origin = 'http://127.0.0.1:' + server.address().port;
	const child = spawn(chrome, ['--headless', '--disable-gpu', '--no-first-run', '--no-default-browser-check', '--disable-background-networking',
		'--remote-debugging-port=0', '--user-data-dir=' + profile, 'about:blank'], {stdio: ['ignore', 'ignore', 'pipe'], detached: process.platform !== 'win32'});
	const exited = once(child, 'exit').catch(() => {});
	let cdp;
	t.after(async () => {
		// Browser.close lets renderers flush and stop before removing their profile.
		// Killing only the launcher can leave Linux Chrome descendants writing files.
		try { cdp?.shutdown(); } catch {}
		let timer;
		try {
			await Promise.race([exited, new Promise(resolve => { timer = setTimeout(resolve, 3000); })]);
		} finally {
			clearTimeout(timer);
			if (process.platform !== 'win32') {
				try { process.kill(-child.pid, 'SIGKILL'); } catch (error) { if (error.code !== 'ESRCH') throw error; }
			} else if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL');
			await exited;
			cdp?.close();
			server.closeAllConnections();
			await new Promise(resolve => server.close(resolve));
			await fs.promises.rm(profile, {recursive: true, force: true, maxRetries: 10, retryDelay: 100});
		}
	});
	const address = await new Promise((resolve, reject) => {
		let stderr = '';
		// Fresh hosted runners can spend tens of seconds initializing Chrome.
		const timer = setTimeout(() => reject(new Error('Chrome did not start: ' + stderr)), 45000);
		const onExit = code => { clearTimeout(timer); reject(new Error('Chrome exited before starting (' + code + '): ' + stderr)); };
		child.once('exit', onExit);
		child.once('error', error => { clearTimeout(timer); reject(error); });
		child.stderr.on('data', chunk => {
			stderr += chunk;
			const match = stderr.match(/DevTools listening on (ws:\/\/[^\s]+)/);
			if (match) { clearTimeout(timer); child.removeListener('exit', onExit); resolve(match[1]); }
		});
	});
	const debugOrigin = new URL(address).origin.replace('ws:', 'http:');
	async function connection(target) {
		const socket = new WebSocket(target);
		await once(socket, 'open');
		let seq = 0;
		const calls = new Map();
		socket.addEventListener('message', event => {
			const message = JSON.parse(event.data);
			const pending = calls.get(message.id);
			if (pending) {
				clearTimeout(pending.timer); calls.delete(message.id);
				if (message.error) pending.reject(new Error(JSON.stringify(message.error))); else pending.resolve(message.result);
			}
		});
		return {
			close: () => socket.close(),
			shutdown: () => socket.send(JSON.stringify({id: ++seq, method: 'Browser.close'})),
			send: (method, params = {}) => new Promise((resolve, reject) => {
				const id = ++seq;
				const timer = setTimeout(() => { calls.delete(id); reject(new Error('CDP timeout: ' + method)); }, 20000);
				calls.set(id, {resolve, reject, timer}); socket.send(JSON.stringify({id, method, params}));
			})
		};
	}
	// The debugger socket can start listening before the initial page exists.
	let target;
	const targetDeadline = Date.now() + 15000;
	while (!target && Date.now() < targetDeadline) {
		const tabs = await (await fetch(debugOrigin + '/json/list')).json();
		target = tabs.find(tab => tab.type === 'page');
		if (!target) await new Promise(resolve => setTimeout(resolve, 25));
	}
	if (!target) throw new Error('Chrome did not create its initial page');
	cdp = await connection(target.webSocketDebuggerUrl);
	await cdp.send('Runtime.enable');
	await cdp.send('Page.enable');
	await cdp.send('Emulation.setDeviceMetricsOverride', {width: options.width || 1440, height: options.height || 1100, deviceScaleFactor: 1, mobile: false});
	async function evaluate(expression) {
		const result = await cdp.send('Runtime.evaluate', {expression, awaitPromise: true, returnByValue: true});
		if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description || result.exceptionDetails.text);
		return result.result.value;
	}
	async function wait(expression) {
		const deadline = Date.now() + 15000;
		while (Date.now() < deadline) {
			if (await evaluate(expression)) return;
			await new Promise(resolve => setTimeout(resolve, 25));
		}
		throw new Error('Browser condition timed out: ' + expression);
	}
	async function navigate(route) {
		await cdp.send('Page.navigate', {url: origin + route});
		await wait('location.pathname === ' + JSON.stringify(new URL(origin + route).pathname) + ' && document.readyState === "complete" && Boolean(window.AutoDocs) && Boolean(document.getElementById("app-navigation")?.childElementCount)');
	}
	await navigate((options.route || '/autodocs-dashboard-page') + (options.query || ''));
	return {
		origin, requests, evaluate, wait, navigate, send: cdp.send,
		reload: async () => {
			const previous = await evaluate('window.__fixtureLoad');
			await cdp.send('Page.reload');
			await wait('window.__fixtureLoad !== ' + JSON.stringify(previous) + ' && document.readyState === "complete" && Boolean(window.AutoDocs) && Boolean(document.getElementById("app-navigation")?.childElementCount)');
		},
		newTab: async route => {
			const target = await (await fetch(debugOrigin + '/json/new?' + encodeURIComponent(origin + route), {method: 'PUT'})).json();
			return connection(target.webSocketDebuggerUrl);
		},
		screenshot: async filename => {
			const result = await cdp.send('Page.captureScreenshot', {format: 'png', captureBeyondViewport: true});
			fs.mkdirSync(path.dirname(filename), {recursive: true}); fs.writeFileSync(filename, Buffer.from(result.data, 'base64'));
		}
	};
}
module.exports = {browserFixture, chrome};
