/* ConfigMap content and live directory favourites share the overview. */
(() => {
	'use strict';
	const byID = id => document.getElementById(id);
	const {element, icon} = AutoDocs;
	const model = DirectoryModel;
	let sources = [], loaded = false;
	const hasStars = () => {
		const prefs = AutoDocs.preferences.directory;
		return prefs.linkStars.length > 0 || prefs.groupStars.length > 0;
	};
	async function request(path) {
		const response = await fetch(path, {cache: 'no-store', signal: AbortSignal.timeout(12000)});
		if (!response.ok) throw new Error('Request failed (' + response.status + ')');
		return response;
	}
	async function loadContent() {
		try {
			const config = await (await request('/api/v1/home')).json();
			if (!config || typeof config.title !== 'string') throw new Error('Invalid configuration');
			for (const key of ['title', 'description', 'environment', 'footer']) byID('home-' + key).textContent = config[key] || '';
			document.title = config.title + ' · AutoDocs';
			byID('home-motd').textContent = config.motd || '';
			byID('motd-panel').hidden = !config.motd;
			byID('home-config-notice').hidden = true;
		} catch {
			byID('home-config-notice').textContent = 'Home page content could not be updated. Showing the last loaded content or defaults.';
			byID('home-config-notice').hidden = false;
		}
	}
	function renderFavourites() {
		const prefs = AutoDocs.preferences.directory;
		const {rows} = model.project(model.index(sources, prefs), prefs, {query: '', namespace: '', kind: '', favourites: true});
		const list = byID('home-favourites');
		list.replaceChildren();
		for (const item of rows) {
			const card = element('article', 'favourite-card');
			card.append(element('h3', 'favourite-name', item.resource.name), element('p', 'favourite-context muted mono', [item.resource.clusterName || item.resource.clusterId, item.resource.namespace, item.resource.kind, ...Object.values(item.resource.classification || {}).map(value => value.display)].filter(Boolean).join(' / ')));
			for (const endpoint of item.endpoints) {
				const link = element('a', 'favourite-link');
				link.href = endpoint.url;
				link.target = '_blank';
				link.rel = 'noopener noreferrer';
				link.append(icon('star'), element('span', '', endpoint.url));
				card.append(link);
			}
			if (!item.endpoints.length) card.append(element('span', 'muted', 'No resolved URL'));
			list.append(card);
		}
		byID('favourites-empty').hidden = rows.length > 0;
		byID('favourites-empty').textContent = hasStars()
			? loaded ? 'No saved favourites are currently available in the directory.' : 'Loading favourites…'
			: 'Star links or groups in the directory to see them here.';
	}
	let loadingFavourites = false;
	async function loadFavourites() {
		if (loadingFavourites) return;
		if (!hasStars()) { byID('favourites-notice').hidden = true; renderFavourites(); return; }
		loadingFavourites = true;
		try {
			const results = await Promise.allSettled(['/api/v1/auto-docs-dashboard', '/api/v1/auto-docs-dashboard?view=discovered'].map(async path => {
				const response = await request(path);
				const data = await response.json();
				if (!Array.isArray(data)) throw new Error('Invalid directory');
				return {data, warnings: response.headers.get('X-AutoDocs-Warnings')};
			}));
			const available = results.filter(result => result.status === 'fulfilled');
			const unique = new Map(available.flatMap(result => result.value.data).map(resource => [model.sourceID(resource), resource]));
			sources = [...unique.values()];
			loaded = true;
			const partial = available.length !== results.length || available.some(result => {
				try { return JSON.parse(result.value.warnings || '[]').length > 0; } catch { return true; }
			});
			byID('favourites-notice').hidden = !partial;
			byID('favourites-notice').textContent = 'Some favourites could not be loaded. The directory may have more details.';
			renderFavourites();
		} finally { loadingFavourites = false; }
	}
	document.addEventListener('DOMContentLoaded', () => {
		const refresh = () => Promise.allSettled([loadContent(), loadFavourites()]);
		window.addEventListener('inventory-scope', renderFavourites);
		window.addEventListener('inventory-generation', loadFavourites);
		AutoDocs.subscribe(() => {
			renderFavourites();
			if (hasStars() && !loaded) loadFavourites();
		});
		renderFavourites();
		refresh();
		setInterval(() => { if (!document.hidden) refresh(); }, 60000);
		document.addEventListener('visibilitychange', () => { if (!document.hidden) refresh(); });
	});
})();
