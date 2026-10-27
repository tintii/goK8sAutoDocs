let allData = [];
let activeView = 'published';
let requestVersion = 0;
let directoryReady = false;
const groupLabel = 'k8sautodocs.io/group';

function element(tag, className, text) {
	const node = document.createElement(tag);
	if (className) node.className = className;
	if (text !== undefined) node.textContent = text;
	return node;
}

function updateThemeButton(theme) {
	document.getElementById('theme-toggle').textContent = theme === 'dark' ? '☀️ Light Mode' : '🌙 Dark Mode';
}

function searchableValues(resource) {
	return [resource.name, resource.namespace, resource.kind, resource['ingress-class'], resource.labels?.[groupLabel],
		...(resource.hostnames || []).map(host => host.host),
		...(resource.endpoints || []).flatMap(endpoint => [endpoint.url, endpoint.listener, endpoint.gateway?.name, endpoint.gateway?.namespace]),
		...(resource.relatedResources || []).flatMap(related => [related.name, related.namespace, related.kind, ...Object.values(related.labels || {})])];
}

function performSearch(searchTerm) {
	if (!directoryReady) return;
	const query = searchTerm.trim().toLowerCase();
	const filtered = allData.filter(resource => searchableValues(resource).some(value => String(value || '').toLowerCase().includes(query)));
	updateDisplay(filtered);
	document.getElementById('filtered-count').textContent = query ? `Showing ${filtered.length} of ${allData.length} resources` : 'Showing all results';
	document.getElementById('clear-search').style.display = query ? 'inline-block' : 'none';
}

function resourceCard(resource) {
	const card = element('div', 'endpoint-button');
	card.appendChild(element('div', 'endpoint-name', resource.name));
	card.appendChild(element('div', 'endpoint-class', resource.kind));
	card.appendChild(element('div', 'endpoint-context', `Namespace: ${resource.namespace}`));
	if (resource['ingress-class']) card.appendChild(element('div', 'endpoint-context', `Ingress class: ${resource['ingress-class']}`));
	for (const endpoint of resource.endpoints || []) {
		const link = element('a', 'endpoint-link', endpoint.url);
		link.href = endpoint.url;
		link.target = '_blank';
		link.rel = 'noopener noreferrer';
		card.appendChild(link);
		if (endpoint.gateway) {
			card.appendChild(element('div', 'endpoint-context', `Gateway ${endpoint.gateway.namespace}/${endpoint.gateway.name} · ${endpoint.listener}`));
		}
	}
	if (!(resource.endpoints || []).length) {
		card.appendChild(element('div', 'endpoint-host', 'No concrete endpoint URL available'));
	}
	if ((resource.issues || []).length) {
		const issues = element('ul', 'endpoint-issues');
		for (const issue of resource.issues) issues.appendChild(element('li', '', issue));
		card.appendChild(issues);
	}
	if ((resource.relatedResources || []).length) {
		const details = element('details', 'related-resources');
		details.appendChild(element('summary', 'related-resources-toggle', `Related resources (${resource.relatedResources.length})`));
		for (const related of resource.relatedResources) {
			const item = element('div', 'related-resource-item');
			item.appendChild(element('span', 'related-resource-name', `${related.namespace}/${related.name}`));
			item.appendChild(element('span', 'related-resource-kind', related.kind));
			details.appendChild(item);
		}
		card.appendChild(details);
	}
	return card;
}

function updateDisplay(data) {
	const container = document.getElementById('content-container');
	container.replaceChildren();
	if (!data.length) {
		const subject = activeView === 'discovered' ? 'discovered' : 'published';
		container.appendChild(element('p', 'empty-directory', allData.length ? `No matching ${subject} resources.` : `No ${subject} resources found.`));
		return;
	}
	const ordered = [...data].sort((a, b) => {
		for (const key of ['namespace', 'kind', 'name']) {
			const order = compareText(a[key] || '', b[key] || '');
			if (order) return order;
		}
		return 0;
	});
	if (activeView === 'discovered') {
		const grid = element('div', 'endpoint-grid');
		for (const resource of ordered) grid.appendChild(resourceCard(resource));
		container.appendChild(grid);
		return;
	}
	const groups = new Map();
	for (const resource of ordered) {
		const group = resource.labels?.[groupLabel] || null;
		if (!groups.has(group)) groups.set(group, []);
		groups.get(group).push(resource);
	}
	const groupNames = [...groups.keys()].filter(group => group !== null).sort(compareText);
	if (groups.has(null)) groupNames.push(null);
	for (const group of groupNames) {
		const section = element('section', 'namespace-section');
		section.appendChild(element('h2', 'namespace-header', group === null ? 'Ungrouped' : group));
		if (group === null) section.appendChild(element('p', 'endpoint-context', 'Published applications without a group.'));
		const grid = element('div', 'endpoint-grid');
		for (const resource of groups.get(group)) grid.appendChild(resourceCard(resource));
		section.appendChild(grid);
		container.appendChild(section);
	}
}

function compareText(a, b) {
	return a < b ? -1 : a > b ? 1 : 0;
}

function viewFromURL() {
	const values = new URLSearchParams(window.location.search).getAll('view');
	return values.length === 1 && values[0] === 'discovered' ? 'discovered' : 'published';
}

function selectView(view, pushHistory = true) {
	activeView = view === 'discovered' ? 'discovered' : 'published';
	if (pushHistory) {
		const url = new URL(window.location.href);
		url.searchParams.set('view', activeView);
		if (url.href !== window.location.href) window.history.pushState(null, '', url);
	}
	for (const tab of document.querySelectorAll('[role="tab"]')) {
		const selected = tab.dataset.view === activeView;
		tab.setAttribute('aria-selected', String(selected));
		tab.tabIndex = selected ? 0 : -1;
	}
	document.getElementById('directory-panel').setAttribute('aria-labelledby', `${activeView}-tab`);
	document.getElementById('resource-count-label').textContent = activeView === 'discovered' ? 'Discovered Resources' : 'Published Resources';
	document.getElementById('directory-api-link').href = `/api/v1/auto-docs-dashboard?view=${activeView}`;
	return loadDirectory();
}

async function loadDirectory() {
	const version = ++requestVersion;
	const view = activeView;
	directoryReady = false;
	allData = [];
	document.getElementById('content-container').replaceChildren();
	document.getElementById('content-container').style.display = 'none';
	const warningBox = document.getElementById('collection-warnings');
	warningBox.hidden = true;
	warningBox.textContent = '';
	const loading = document.getElementById('loading');
	loading.style.display = 'block';
	loading.textContent = `Loading ${view === 'discovered' ? 'discovered endpoints' : 'applications'}...`;
	for (const id of ['total-resources', 'total-namespaces', 'total-hosts']) document.getElementById(id).textContent = '-';
	document.getElementById('filtered-count').textContent = 'Loading results...';
	try {
		const response = await fetch(`/api/v1/auto-docs-dashboard?view=${view}`);
		if (version !== requestVersion) return;
		if (!response.ok) throw new Error(`Request failed (${response.status})`);
		const warnings = JSON.parse(response.headers.get('X-AutoDocs-Warnings') || '[]');
		const data = await response.json();
		if (version !== requestVersion) return;
		if (!Array.isArray(data)) throw new Error('Unexpected directory response');
		warningBox.hidden = !warnings.length;
		warningBox.textContent = warnings.length ? `Discovery incomplete: ${warnings.join('; ')}` : '';
		allData = data;
		directoryReady = true;
		document.getElementById('total-resources').textContent = data.length;
		document.getElementById('total-namespaces').textContent = new Set(data.map(resource => resource.namespace)).size;
		document.getElementById('total-hosts').textContent = data.reduce((sum, resource) => sum + (resource.endpoints || []).length, 0);
		document.getElementById('loading').style.display = 'none';
		document.getElementById('content-container').style.display = 'block';
		performSearch(document.getElementById('search-input').value);
	} catch (error) {
		if (version !== requestVersion) return;
		loading.textContent = `Error loading ${view === 'discovered' ? 'Discovered' : 'Applications'}: ${error.message}`;
		document.getElementById('filtered-count').textContent = 'Results unavailable';
	}
}

document.addEventListener('DOMContentLoaded', () => {
	const tabs = [...document.querySelectorAll('[role="tab"]')];
	for (const tab of tabs) {
		tab.addEventListener('click', () => selectView(tab.dataset.view));
		tab.addEventListener('keydown', event => {
			let index = tabs.indexOf(tab);
			if (event.key === 'ArrowRight') index = (index + 1) % tabs.length;
			else if (event.key === 'ArrowLeft') index = (index + tabs.length - 1) % tabs.length;
			else if (event.key === 'Home') index = 0;
			else if (event.key === 'End') index = tabs.length - 1;
			else return;
			event.preventDefault();
			tabs[index].focus();
			selectView(tabs[index].dataset.view);
		});
	}
	window.addEventListener('popstate', () => selectView(viewFromURL(), false));
	const theme = localStorage.getItem('theme') || 'light';
	document.documentElement.setAttribute('data-theme', theme);
	updateThemeButton(theme);
	document.getElementById('theme-toggle').addEventListener('click', () => {
		const nextTheme = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
		document.documentElement.setAttribute('data-theme', nextTheme);
		localStorage.setItem('theme', nextTheme);
		updateThemeButton(nextTheme);
	});
	const search = document.getElementById('search-input');
	let searchTimeout;
	search.addEventListener('input', () => {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => performSearch(search.value), 300);
	});
	const clearSearch = () => {
		search.value = '';
		performSearch('');
	};
	document.getElementById('clear-search').addEventListener('click', clearSearch);
	search.addEventListener('keydown', event => {
		if (event.key === 'Escape') clearSearch();
	});
	fetch('/api/v1/cluster-info')
		.then(response => {
			if (!response.ok) throw new Error('Cluster information unavailable');
			return response.json();
		})
		.then(data => { document.getElementById('cluster-name').textContent = data.clusterName; })
		.catch(() => { document.getElementById('cluster-name').textContent = 'Cluster: Unknown'; });
	selectView(viewFromURL(), false);
});
