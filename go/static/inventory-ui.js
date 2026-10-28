/* Shared inventory scope and refresh state. Polls AutoDocs, never Kubernetes. */
(() => {
	'use strict';
	const {element, button} = AutoDocs;
	const dimensions = ['cluster', 'environment', 'datacentre'];
	const unknown = '\u0000';
	const readScope = () => Object.fromEntries(dimensions.map(key => [key, new URL(location.href).searchParams.get(key) || '']));
	let scope = readScope(), status = null, offset = 0, busy = false, sending = false, error = '', timer;
	const selectors = new Map();
	const validTime = value => value && Number.isFinite(Date.parse(value)) && Date.parse(value) > 0;
	const now = () => Date.now() + offset;
	const age = value => { const seconds = Math.max(0, Math.floor((now() - Date.parse(value)) / 1000)); return seconds < 5 ? 'just now' : seconds < 60 ? seconds + 's ago' : seconds < 3600 ? Math.floor(seconds / 60) + 'm ago' : Math.floor(seconds / 3600) + 'h ago'; };
	const countdown = value => { const seconds = Math.max(0, Math.ceil((Date.parse(value) - now()) / 1000)); return seconds >= 60 ? Math.floor(seconds / 60) + 'm ' + seconds % 60 + 's' : seconds + 's'; };
	function matches(resource) {
		if (scope.cluster && resource.clusterId !== scope.cluster) return false;
		return ['environment', 'datacentre'].every(key => !scope[key] || (resource.classification?.[key]?.id ?? unknown) === scope[key]);
	}
	function links() {
		for (const link of document.querySelectorAll('a.nav-item, a.brand, a.destination, .home-favourites-heading > a')) {
			const url = new URL(link.href, location.href);
			if (url.origin !== location.origin || !['/', '/autodocs-dashboard-page', '/versions-dashboard-page'].includes(url.pathname)) continue;
			for (const key of dimensions) { if (scope[key]) url.searchParams.set(key, scope[key]); else url.searchParams.delete(key); }
			link.href = url.pathname + url.search;
		}
	}
	function setScope(next, push = true) {
		scope = {...scope, ...next};
		if (push) { const url = new URL(location.href); for (const key of dimensions) { if (scope[key]) url.searchParams.set(key, scope[key]); else url.searchParams.delete(key); } if (url.href !== location.href) history.pushState(null, '', url); }
		options(); links(); render(); window.dispatchEvent(new Event('inventory-scope'));
	}
	function options() {
		for (const [key, select] of selectors) {
			const items = key === 'cluster' ? (status?.clusters || []).map(target => [target.id, target.name]) : Object.entries(status?.dimensions?.[key] || {});
			if (key !== 'cluster') items.push([unknown, key === 'environment' ? 'Unclassified' : 'Unspecified']);
			if (scope[key] && !items.some(([id]) => id === scope[key])) items.push([scope[key], scope[key] + ' (unavailable)']);
			select.replaceChildren(new Option('All ' + (key === 'cluster' ? 'clusters' : key === 'environment' ? 'environments' : 'datacentres'), ''));
			for (const [id, label] of items.sort((a, b) => a[1].localeCompare(b[1]))) select.append(new Option(label, id));
			select.value = scope[key];
		}
	}
	function relevantParts() {
		const keys = location.pathname === '/versions-dashboard-page' ? ['workloads'] : ['ingresses', 'httproutes'];
		return (status?.clusters || []).filter(target => !scope.cluster || scope.cluster === target.id).flatMap(target => keys.map(key => ({target, key, part: target.parts[key]}))).filter(({part}) => part && ['environment', 'datacentre'].every(key => !scope[key] || part.values?.[key]?.[scope[key]]));
	}
	function render() {
		const label = document.getElementById('inventory-state'), update = document.getElementById('inventory-update');
		if (!label) return;
		if (!status) { label.textContent = error || 'Loading inventory status…'; update.disabled = true; return; }
		const parts = relevantParts();
		const times = parts.map(({part}) => part.lastSuccessAt).filter(validTime).sort((a, b) => Date.parse(a) - Date.parse(b));
		const stale = new Set(parts.filter(({part}) => part.error || !part.lastSuccessAt).map(({target}) => target.id));
		let summary = status.running ? 'Updating ' + status.completed + ' / ' + status.total + ' clusters…' : times.length ? 'Updated ' + age(times[0]) : 'No collected inventory in this scope';
		if (stale.size) summary += ' · ' + stale.size + ' cluster' + (stale.size === 1 ? '' : 's') + ' incomplete or stale';
		if (!status.running && validTime(status.nextAutomaticAt)) summary += ' · Next attempt in ' + countdown(status.nextAutomaticAt);
		if (error || status.cacheError) summary += ' · ' + (error || status.cacheError);
		label.textContent = summary;
		const cooling = validTime(status.nextManualAt) && Date.parse(status.nextManualAt) > now();
		update.disabled = sending || status.running || cooling;
		update.textContent = status.running ? 'Updating…' : cooling ? 'Update in ' + countdown(status.nextManualAt) : 'Update now';
		const directoryUpdate = document.getElementById('refresh-directory');
		if (directoryUpdate) { directoryUpdate.disabled = update.disabled; directoryUpdate.title = update.textContent + ' · shared by all viewers'; }
		const details = document.getElementById('inventory-details-list'); details.replaceChildren();
		for (const {target, key, part} of parts) details.append(element('li', part.error ? 'status-attention' : '', target.name + ' / ' + key + ': ' + (part.lastSuccessAt ? 'updated ' + age(part.lastSuccessAt) : 'never collected') + (part.error ? ' · ' + part.error : '') + (part.warnings?.length ? ' · ' + part.warnings.join('; ') : '')));
	}
	async function accept(response) {
		if (!response.ok) throw new Error('Inventory status unavailable');
		const next = await response.json();
		if (!Array.isArray(next.clusters) || !Number.isFinite(next.generation) || !validTime(next.serverTime)) throw new Error('Inventory status unavailable');
		const previous = status?.generation;
		status = next; offset = Date.parse(next.serverTime) - Date.now(); error = '';
		AutoDocs.migrateCluster(next.legacyClusterID);
		options(); render();
		if (previous !== undefined && previous !== next.generation) window.dispatchEvent(new Event('inventory-generation'));
	}
	async function poll() {
		if (busy || document.hidden) return;
		busy = true;
		try { await accept(await fetch('/api/v1/discovery-status', {cache:'no-store', signal:AbortSignal.timeout(10000)})); }
		catch { error = 'Inventory status unavailable'; render(); }
		finally { busy = false; clearTimeout(timer); timer = setTimeout(poll, status?.running ? 2000 : 15000); }
	}
	async function refresh() {
		if (sending) return;
		sending = true; render();
		try { await accept(await fetch('/api/v1/discovery-refresh', {method:'POST', headers:{'Content-Type':'application/json'}, cache:'no-store', body:JSON.stringify({generation:status?.generation}), signal:AbortSignal.timeout(10000)})); }
		catch { error = 'Update could not be requested'; }
		finally { sending = false; render(); clearTimeout(timer); timer = setTimeout(poll, 1000); }
	}
	function init() {
		const bar = element('section', 'inventory-toolbar'); bar.setAttribute('aria-label', 'Inventory scope and updates');
		const filters = element('div', 'filters inventory-filters');
		for (const key of dimensions) { const label = element('label', 'filter-group', key[0].toUpperCase() + key.slice(1) + ' '), select = element('select'); select.id = 'scope-' + key; label.append(select); filters.append(label); selectors.set(key, select); select.addEventListener('change', () => setScope({[key]:select.value})); }
		const row = element('div', 'inventory-freshness'), label = element('span', 'muted'); label.id = 'inventory-state'; label.setAttribute('role', 'status');
		const update = button('Update now', refresh, 'button quiet'); update.id = 'inventory-update'; row.append(label, update);
		const details = element('details', 'inventory-details'); details.append(element('summary', '', 'Collection details')); const list = element('ul'); list.id = 'inventory-details-list'; details.append(list);
		bar.append(filters, row, details); document.getElementById('main').insertBefore(bar, document.querySelector('main section'));
		options(); links(); poll(); setInterval(() => { if (!document.hidden) render(); }, 1000);
		window.addEventListener('popstate', () => setScope(readScope(), false));
		document.addEventListener('visibilitychange', () => { if (!document.hidden) poll(); });
	}
	window.Inventory = {matches, refresh, setScope, get scope() { return {...scope}; }, get status() { return status; }};
	document.addEventListener('DOMContentLoaded', init);
})();
