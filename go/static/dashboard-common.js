/* Shared preferences and shell. No cluster data is persisted. */
(() => {
	'use strict';
	const key = 'k8sautodocs.preferences';
	const modes = ['label', 'namespace', 'kind', 'custom', 'none'];
	const object = value => value !== null && typeof value === 'object' && !Array.isArray(value);
	const strings = value => Array.isArray(value) && value.every(item => typeof item === 'string');
	const defaultDirectory = () => ({groupBy: 'label', density: 'grid', collapsed: [], groups: [], assignments: {}, groupStars: [], linkStars: []});
	const versionColumns = ['resource', 'namespace', 'kind', 'container', 'type', 'image', 'tag', 'sha'];
	const defaultVersions = () => ({widths: {}, sort: []});
	const defaults = () => ({version: 1, theme: null, directory: defaultDirectory(), versions: defaultVersions()});
	let state = defaults();
	let notice = '';
	let future = false;
	const listeners = new Set();

	function validateVersions(value) {
		if (!object(value) || !object(value.widths) || !Array.isArray(value.sort) || value.sort.length > 2) return defaultVersions();
		const keys = new Set();
		for (const sort of value.sort) {
			if (!object(sort) || !versionColumns.includes(sort.key) || !['asc', 'desc'].includes(sort.direction) || keys.has(sort.key)) return defaultVersions();
			keys.add(sort.key);
		}
		for (const [key, width] of Object.entries(value.widths)) {
			if (!versionColumns.includes(key) || !Number.isFinite(width) || width < 64 || width > 1200) return defaultVersions();
		}
		return {widths: {...value.widths}, sort: value.sort.map(({key, direction}) => ({key, direction}))};
	}
	function validate(value) {
		if (!object(value) || value.version !== 1) throw new Error('Unsupported preferences');
		if (![null, 'light', 'dark'].includes(value.theme)) throw new Error('Invalid theme');
		const d = value.directory;
		if (!object(d) || !modes.includes(d.groupBy) || !['grid', 'list'].includes(d.density) ||
			!strings(d.collapsed) || !strings(d.groupStars) || !strings(d.linkStars) || !Array.isArray(d.groups) ||
			!object(d.assignments)) throw new Error('Invalid directory preferences');
		const ids = new Set(), names = new Set();
		for (const group of d.groups) {
			if (!object(group) || typeof group.id !== 'string' || !group.id || typeof group.name !== 'string' ||
				!group.name.trim() || group.name !== group.name.trim() || ids.has(group.id) || names.has(group.name)) throw new Error('Invalid custom groups');
			ids.add(group.id); names.add(group.name);
		}
		for (const [source, group] of Object.entries(d.assignments)) {
			let tuple;
			try { tuple = JSON.parse(source); } catch { throw new Error('Invalid source identity'); }
			if (!strings(tuple) || tuple.length !== 3 || !ids.has(group)) throw new Error('Invalid membership');
		}
		// Copy only supported fields, including safe own properties for arbitrary source keys.
		return {version: 1, theme: value.theme, versions: validateVersions(value.versions), directory: {
			groupBy: d.groupBy, density: d.density, collapsed: [...new Set(d.collapsed)],
			groups: d.groups.map(g => ({id: g.id, name: g.name})),
			assignments: Object.fromEntries(Object.entries(d.assignments)),
			groupStars: [...new Set(d.groupStars)], linkStars: [...new Set(d.linkStars)]
		}};
	}
	function decode(raw) {
		if (raw === null) return defaults();
		const parsed = JSON.parse(raw);
		if (object(parsed) && typeof parsed.version === 'number' && parsed.version > 1) {
			future = true;
			throw new Error('Future version');
		}
		return validate(parsed);
	}
	try {
		const raw = localStorage.getItem(key);
		state = decode(raw);
		if (raw === null) {
			const legacy = localStorage.getItem('theme');
			if (['light', 'dark'].includes(legacy)) state.theme = legacy;
		}
	} catch {
		notice = future ? 'These preferences use a newer version. Changes apply to this session only.' :
			'Browser preferences could not be read. Using defaults; changes may not survive reload.';
	}
	const media = window.matchMedia('(prefers-color-scheme: dark)');
	function theme() { return state.theme || (media.matches ? 'dark' : 'light'); }
	function apply() {
		document.documentElement.dataset.theme = theme();
		const toggle = document.getElementById('theme-toggle');
		if (toggle) {
			toggle.replaceChildren(icon(theme() === 'dark' ? 'sun' : 'moon'), document.createTextNode(theme() === 'dark' ? 'Light mode' : 'Dark mode'));
			toggle.setAttribute('aria-label', 'Switch to ' + (theme() === 'dark' ? 'light' : 'dark') + ' theme');
		}
		const box = document.getElementById('preference-notice');
		if (box) { box.textContent = notice; box.hidden = !notice; }
	}
	function emit() { apply(); for (const listener of listeners) listener(); }
	function update(change) {
		const next = structuredClone(state);
		change(next);
		state = validate(next);
		if (!future) {
			try { localStorage.setItem(key, JSON.stringify(state)); notice = ''; }
			catch { notice = 'Browser storage is unavailable or full. Changes apply to this session only.'; }
		}
		emit();
	}
	window.addEventListener('storage', event => {
		if (event.key !== key && event.key !== null) return;
		try {
			future = false;
			const next = decode(event.newValue);
			state = next; notice = '';
		} catch {
			notice = future ? 'These preferences use a newer version. Changes apply to this session only.' : 'Another tab saved unreadable preferences. Keeping this session’s settings.';
		}
		emit();
	});
	media.addEventListener('change', () => { if (!state.theme) apply(); });

	function element(tag, className, text) {
		const node = document.createElement(tag);
		if (className) node.className = className;
		if (text !== undefined) node.textContent = text;
		return node;
	}
	const paths = {
		grid: 'M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z',
		home: 'm3 10 9-7 9 7v11h-6v-7H9v7H3z',
		layers: 'm12 3 10 5-10 5L2 8z M2 12l10 5 10-5 M2 16l10 5 10-5',
		star: 'm12 3 2.8 5.7 6.2.9-4.5 4.4 1.1 6.2-5.6-3-5.6 3 1.1-6.2L3 9.6l6.2-.9z',
		search: 'M21 21l-5-5 M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0',
		arrow: 'M5 12h14 m-6-6 6 6-6 6',
		external: 'M14 3h7v7 M21 3 10 14 M10 3H3v18h18v-7',
		refresh: 'M20 8a9 9 0 1 0 1 8 M20 3v6h-6',
		settings: 'M4 7h16 M4 17h16 M8 4v6 M16 14v6',
		sun: 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M12 1v3 M12 20v3 M1 12h3 M20 12h3 M4 4l2 2 M18 18l2 2 M4 20l2-2 M18 6l2-2',
		moon: 'M21 13A9 9 0 0 1 11 3 9 9 0 1 0 21 13',
		code: 'm8 5-7 7 7 7 M16 5l7 7-7 7 M14 3l-4 18',
		list: 'M8 5h13 M8 12h13 M8 19h13 M2 5h1 M2 12h1 M2 19h1',
		chevron: 'm9 5 7 7-7 7',
		close: 'm6 6 12 12 M6 18 18 6',
		globe: 'M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0 M3 12h18 M12 3a18 18 0 0 1 0 18 18 18 0 0 1 0-18',
		box: 'm12 3 9 5v9l-9 5-9-5V8z M3 8l9 5 9-5 M12 13v9',
	};
	function icon(name) {
		const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
		for (const [key, value] of Object.entries({viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '1.6', 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'aria-hidden': 'true'})) svg.setAttribute(key, value);
		const path = document.createElementNS(svg.namespaceURI, 'path'); path.setAttribute('d', paths[name] || paths.box);
		svg.appendChild(path); return svg;
	}
	function button(text, action, className = 'button', iconName) {
		const node = element('button', className); node.type = 'button';
		if (iconName) node.appendChild(icon(iconName));
		node.appendChild(document.createTextNode(text)); node.addEventListener('click', action); return node;
	}
	function initShell() {
		for (const node of document.querySelectorAll('[data-icon]')) {
			if (node.className) node.replaceChildren(icon(node.dataset.icon));
			else node.replaceWith(icon(node.dataset.icon));
		}
		const host = document.getElementById('app-navigation');
		if (host) {
			const brand = element('a', 'brand'); brand.href = '/';
			brand.append(icon('layers'), element('span', '', 'AutoDocs'), element('span', 'brand-dot', ''));
			const nav = element('nav', 'main-nav'); nav.setAttribute('aria-label', 'Main navigation');
			for (const [href, label, symbol] of [['/', 'Overview', 'home'], ['/autodocs-dashboard-page', 'Directory', 'grid'], ['/versions-dashboard-page', 'Versions', 'layers']]) {
				const link = element('a', 'nav-item'); link.href = href;
				link.append(icon(symbol), document.createTextNode(label));
				if (window.location.pathname === href) link.setAttribute('aria-current', 'page');
				nav.appendChild(link);
			}
			const foot = element('div', 'rail-footer');
			foot.append(element('div', 'rail-mark', 'YOUR CLUSTER. CONNECTED.'), element('p', '', 'A little order for everything you run.'));
			host.append(brand, nav, foot);
		}
		const toggle = document.getElementById('theme-toggle');
		if (toggle) toggle.addEventListener('click', () => update(prefs => { prefs.theme = theme() === 'dark' ? 'light' : 'dark'; }));
		apply();
		fetch('/api/v1/cluster-info').then(response => {
			if (!response.ok) throw new Error('Unavailable'); return response.json();
		}).then(data => {
			document.getElementById('cluster-name').textContent = data.clusterName || 'Cluster: Unknown';
		}).catch(() => { document.getElementById('cluster-name').textContent = 'Cluster: Unknown'; });
	}
	window.AutoDocs = {key, get preferences() { return structuredClone(state); }, update, subscribe: listener => listeners.add(listener), defaultDirectory, element, icon, button, validate};
	apply();
	document.addEventListener('DOMContentLoaded', initShell);
})();
