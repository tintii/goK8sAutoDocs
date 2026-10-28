/* Home page content comes from a mounted data file; all content is plain text. */
(() => {
	'use strict';
	const byID = id => document.getElementById(id);
	const node = (tag, className, text) => {
		const el = document.createElement(tag);
		el.className = className;
		el.textContent = text;
		return el;
	};
	async function request(path) {
		const response = await fetch(path, {cache: 'no-store', signal: AbortSignal.timeout(12000)});
		if (!response.ok) throw new Error('Request failed (' + response.status + ')');
		return response.json();
	}
	function safeURL(raw) {
		if (typeof raw !== 'string' || /[\\\r\n\t]/.test(raw)) return false;
		try {
			const url = new URL(raw, location.origin);
			return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password &&
				(/^https?:\/\//.test(raw) || (raw.startsWith('/') && !raw.startsWith('//')));
		} catch { return false; }
	}
	async function loadContent() {
		try {
			const config = await request('/api/v1/home');
			if (!config || typeof config.title !== 'string') throw new Error('Invalid configuration');
			for (const key of ['title', 'description', 'environment', 'footer']) byID('home-' + key).textContent = config[key] || '';
			document.title = config.title + ' · AutoDocs';
			byID('home-motd').textContent = config.motd || '';
			byID('motd-panel').hidden = !config.motd;
			const links = byID('home-links');
			links.replaceChildren();
			for (const link of config.links || []) {
				if (!safeURL(link.url)) continue;
				const anchor = node('a', 'button', link.label);
				anchor.href = link.url;
				anchor.target = '_blank';
				anchor.rel = 'noopener noreferrer';
				links.append(anchor);
			}
			byID('home-links-panel').hidden = !links.childElementCount;
			byID('home-config-notice').hidden = true;
		} catch {
			byID('home-config-notice').textContent = 'Home page content could not be updated. Showing the last loaded content or defaults.';
			byID('home-config-notice').hidden = false;
		}
	}
	async function loadStatus() {
		try {
			const status = await request('/api/v1/service-status');
			if (!Array.isArray(status.workloads)) throw new Error('Invalid status');
			const total = status.workloads.length;
			byID('status-summary').textContent = total === 0 ? 'No workloads found.' : status.attention
				? status.attention + ' workload' + (status.attention === 1 ? ' needs' : 's need') + ' attention.'
				: status.ready ? 'All active workloads are ready.' : 'No active workloads.';
			byID('status-summary').className = status.attention ? 'status-attention' : '';
			for (const key of ['ready', 'attention', 'inactive']) byID('status-' + key).textContent = status[key];
			byID('status-counts').hidden = false;
			const list = byID('workload-statuses');
			list.replaceChildren();
			for (const item of status.workloads.slice(0, 6)) {
				const row = node('li', 'workload-status', '');
				const info = node('div', 'workload-info', '');
				info.append(node('strong', '', item.name), node('span', 'muted mono', item.namespace + ' / ' + item.kind));
				const state = ['ready', 'attention', 'inactive'].includes(item.state) ? item.state : 'attention';
				const labels = {ready: 'Ready', attention: 'Needs attention', inactive: 'Inactive'};
				const availability = node('div', 'workload-availability', '');
				availability.append(node('span', 'muted mono', item.ready + ' / ' + item.desired + ' available'), node('span', 'status-pill status-' + state, labels[state]));
				row.append(info, availability);
				list.append(row);
			}
			byID('status-checked').textContent = 'Updated ' + new Date(status.checkedAt).toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'}) +
				' · ' + (total > 6 ? 'Showing 6 of ' + total + ', attention first. ' : '') + 'Refreshes every 60s. Inactive: scaled to zero or no eligible nodes.';
		} catch {
			byID('status-summary').textContent = 'Workload status unavailable. Check Kubernetes access and read permissions.';
			byID('status-summary').className = 'status-attention';
			byID('status-counts').hidden = true;
			byID('workload-statuses').replaceChildren();
			byID('status-checked').textContent = 'No current status. Retrying every 60 seconds.';
		}
	}
	let loading = false;
	async function refresh() {
		if (loading) return;
		loading = true;
		byID('home-refresh').disabled = true;
		await Promise.allSettled([loadContent(), loadStatus()]);
		byID('home-refresh').disabled = false;
		loading = false;
	}
	document.addEventListener('DOMContentLoaded', () => {
		byID('home-refresh').addEventListener('click', refresh);
		refresh();
		setInterval(() => { if (!document.hidden) refresh(); }, 60000);
		document.addEventListener('visibilitychange', () => { if (!document.hidden) refresh(); });
	});
})();
