let allData = [];
let activeView = 'published';
let requestVersion = 0;
let directoryReady = false;
let directoryIndex = [];
let favouritesOnly = false;
const {element, button, icon} = AutoDocs;
const model = DirectoryModel;
const $ = id => document.getElementById(id);
const directoryPrefs = () => AutoDocs.preferences.directory;
const modifyDirectory = change => AutoDocs.update(prefs => change(prefs.directory));

function preserveFocus(render) {
	const token = document.activeElement?.dataset.control;
	render();
	if (token) [...document.querySelectorAll('[data-control]')].find(node => node.dataset.control === token)?.focus({preventScroll: true});
}
function control(node, token) { node.dataset.control = token; return node; }
function toggleStar(type, id) {
	modifyDirectory(prefs => {
		const list = prefs[type];
		prefs[type] = list.includes(id) ? list.filter(value => value !== id) : [...list, id];
	});
}
function star(type, id, label) {
	const pressed = directoryPrefs()[type].includes(id);
	const node = button('', () => toggleStar(type, id), 'star-button', 'star');
	node.setAttribute('aria-pressed', String(pressed));
	node.setAttribute('aria-label', (pressed ? 'Unstar ' : 'Star ') + label);
	node.title = (pressed ? 'Unstar ' : 'Star ') + label;
	return control(node, type + id);
}
function filters(query) {
	return {query, namespace: $('namespace-filter').value, kind: $('kind-filter').value, favourites: favouritesOnly, view: activeView};
}
function performSearch(searchTerm) {
	$('search-input').value = searchTerm;
	$('clear-search').style.display = searchTerm ? 'inline-flex' : 'none';
	if (!directoryReady) return;
	const projected = model.project(directoryIndex, directoryPrefs(), filters(searchTerm));
	preserveFocus(() => updateDisplay(projected));
	const count = projected.rows.reduce((sum, row) => sum + row.endpoints.length, 0);
	$('filtered-count').textContent = 'Showing ' + projected.rows.length + ' of ' + allData.length + ' resources · ' + count + ' URLs';
}
function resourceCard(row) {
	const resource = row.resource;
	const card = element('article', 'endpoint-button');
	const heading = element('div', 'card-header');
	const monogram = element('div', 'app-monogram', resource.name.slice(0, 2).toUpperCase());
	monogram.setAttribute('aria-hidden', 'true');
	monogram.dataset.tone = String([...resource.name].reduce((sum, char) => sum + char.charCodeAt(0), 0) % 4);
	const title = element('div', 'card-heading-text');
	title.append(element('div', 'endpoint-name', resource.name), element('div', 'endpoint-class', resource.kind));
	heading.append(monogram, title);
	card.appendChild(heading);
	const metadata = element('div', 'card-meta');
	metadata.appendChild(element('span', 'badge', 'Namespace: ' + resource.namespace));
	if (resource.labels?.[model.groupLabel]) metadata.appendChild(element('span', 'badge', resource.labels[model.groupLabel]));
	if (resource['ingress-class']) metadata.appendChild(element('span', 'badge', 'Class: ' + resource['ingress-class']));
	card.appendChild(metadata);
	const links = element('div', 'link-list');
	for (const endpoint of row.endpoints) {
		const linkRow = element('div', 'endpoint-row');
		const link = element('a', 'endpoint-link');
		link.append(element('span', 'link-label mono', endpoint.url.replace(/^https?:\/\//, '')), icon('external'));
		link.href = endpoint.url; link.target = '_blank'; link.rel = 'noopener noreferrer';
		link.setAttribute('aria-label', 'Open ' + resource.name + ': ' + endpoint.url);
		link.title = endpoint.url;
		linkRow.append(link, star('linkStars', model.linkID(resource, endpoint.url), resource.name + ' link ' + endpoint.url));
		links.appendChild(linkRow);
		for (const attachment of endpoint.attachments) {
			links.appendChild(element('div', 'endpoint-context', 'Gateway ' + attachment.gateway.namespace + '/' + attachment.gateway.name + ' · ' + (attachment.listener || '')));
		}
	}
	if (row.endpoints.length) card.appendChild(links);
	else card.appendChild(element('div', 'endpoint-host', 'No concrete endpoint URL available'));
	if (resource.issues?.length) {
		const issues = element('ul', 'endpoint-issues');
		for (const issue of resource.issues) issues.appendChild(element('li', '', issue));
		card.appendChild(issues);
	}
	const footer = element('div', 'card-footer');
	if (resource.relatedResources?.length) {
		const details = element('details', 'related-resources');
		details.appendChild(element('summary', '', 'Related resources (' + resource.relatedResources.length + ')'));
		for (const related of resource.relatedResources) {
			const item = element('div', 'related-resource-item');
			item.append(element('span', 'related-resource-name', related.namespace + '/' + related.name), element('span', 'related-resource-kind', related.kind));
			details.appendChild(item);
		}
		footer.appendChild(details);
	} else footer.appendChild(element('span', 'endpoint-context', 'Source: ' + resource.kind));
	if (activeView === 'published' && directoryPrefs().groups.length) {
		const assignment = element('select', 'assignment');
		assignment.setAttribute('aria-label', 'Personal group for ' + resource.namespace + '/' + resource.name);
		assignment.appendChild(new Option('No personal group', ''));
		for (const group of directoryPrefs().groups) assignment.appendChild(new Option(group.name, group.id));
		assignment.value = directoryPrefs().assignments[row.source] || '';
		assignment.addEventListener('change', () => modifyDirectory(prefs => {
			if (assignment.value) prefs.assignments[row.source] = assignment.value;
			else delete prefs.assignments[row.source];
		}));
		footer.appendChild(control(assignment, 'assign' + row.source));
	}
	card.appendChild(footer);
	return card;
}
function updateDisplay(projected) {
	const container = $('content-container');
	container.replaceChildren();
	if (!projected.rows.length) {
		const empty = element('div', 'empty-state');
		const subject = activeView === 'discovered' ? 'discovered' : 'published';
		empty.appendChild(icon(favouritesOnly ? 'star' : 'search'));
		const message = !allData.length ? 'No ' + subject + ' resources found.' : favouritesOnly ? 'No favourites match this view.' : 'No matching ' + subject + ' resources.';
		empty.append(element('h2', 'empty-directory', message));
		empty.appendChild(element('p', '', !allData.length ? (activeView === 'published' ? 'Published applications will appear here when they are available.' : 'Unlabelled endpoints will appear here when they are discovered.') : 'Try another search, clear your filters or star something you use often.'));
		if (allData.length) empty.appendChild(button('Clear filters', () => {
			favouritesOnly = false; $('namespace-filter').value = ''; $('kind-filter').value = ''; syncControls(); performSearch('');
		}));
		container.appendChild(empty); return;
	}
	const prefs = directoryPrefs();
	for (let i = 0; i < projected.sections.length; i++) {
		const group = projected.sections[i];
		const grid = element('div', 'endpoint-grid'); grid.dataset.density = prefs.density;
		for (const row of group.rows) grid.appendChild(resourceCard(row));
		if (group.id === 'flat') { container.appendChild(grid); continue; }
		const section = element('section', 'namespace-section');
		const heading = element('div', 'group-heading');
		const collapsed = prefs.collapsed.includes(group.id);
		const collapse = control(button('', () => modifyDirectory(value => {
			value.collapsed = collapsed ? value.collapsed.filter(id => id !== group.id) : [...value.collapsed, group.id];
		}), 'button quiet icon-only', 'chevron'), 'collapse' + group.id);
		collapse.setAttribute('aria-expanded', String(!collapsed));
		collapse.setAttribute('aria-label', (collapsed ? 'Expand ' : 'Collapse ') + group.name);
		const gridID = 'group-content-' + i;
		collapse.setAttribute('aria-controls', gridID); grid.id = gridID;
		collapse.firstChild.style.transform = collapsed ? '' : 'rotate(90deg)';
		heading.append(element('span', 'group-index mono', String(i + 1).padStart(2, '0')), element('h2', 'namespace-header', group.name),
			element('span', 'group-count mono', String(group.rows.length)), element('span', 'group-line', ''), element('span', 'group-mode', group.mode === 'custom' ? 'personal' : group.mode),
			star('groupStars', group.id, group.mode + ' group ' + group.name), collapse);
		section.appendChild(heading);
		if (group.fallback) section.appendChild(element('p', 'group-description', group.mode === 'label' ? 'Published applications without a group.' : 'Applications without a personal group.'));
		grid.hidden = collapsed;
		section.appendChild(grid); container.appendChild(section);
	}
}
function syncControls() {
	const prefs = directoryPrefs();
	$('group-by').value = prefs.groupBy;
	$('group-by').disabled = activeView === 'discovered';
	$('density').value = prefs.density;
	$('favourites-toggle').setAttribute('aria-pressed', String(favouritesOnly));
}
function filterOptions(id, key, label) {
	const select = $(id), current = select.value;
	const values = [...new Set(allData.map(resource => resource[key]))].sort();
	if (current && !values.includes(current)) values.push(current);
	select.replaceChildren(new Option(label, ''));
	for (const value of values) select.appendChild(new Option(value, value));
	select.value = current;
}
function viewFromURL() {
	const values = new URLSearchParams(window.location.search).getAll('view');
	return values.length === 1 && values[0] === 'discovered' ? 'discovered' : 'published';
}
function selectView(view, pushHistory = true) {
	activeView = view === 'discovered' ? 'discovered' : 'published';
	if (pushHistory) {
		const url = new URL(window.location.href); url.searchParams.set('view', activeView);
		if (url.href !== window.location.href) window.history.pushState(null, '', url);
	}
	for (const tab of document.querySelectorAll('[role="tab"]')) {
		const selected = tab.dataset.view === activeView;
		tab.setAttribute('aria-selected', String(selected)); tab.tabIndex = selected ? 0 : -1;
	}
	$('directory-panel').setAttribute('aria-labelledby', activeView + '-tab');
	$('resource-count-label').textContent = activeView === 'discovered' ? 'Discovered Resources' : 'Published Resources';
	$('directory-api-link').href = '/api/v1/auto-docs-dashboard?view=' + activeView;
	syncControls();
	return loadDirectory();
}
async function loadDirectory() {
	const version = ++requestVersion, view = activeView;
	directoryReady = false; allData = []; directoryIndex = [];
	$('content-container').replaceChildren(); $('content-container').style.display = 'none';
	$('collection-warnings').hidden = true; $('collection-warnings').textContent = '';
	$('loading').style.display = 'block'; $('loading').textContent = view === 'discovered' ? 'Loading discovered endpoints…' : 'Loading applications…';
	for (const id of ['total-resources', 'total-namespaces', 'total-hosts']) $(id).textContent = '—';
	$('filtered-count').textContent = 'Loading results…';
	try {
		const response = await fetch('/api/v1/auto-docs-dashboard?view=' + view);
		if (version !== requestVersion) return;
		if (!response.ok) throw new Error('Request failed (' + response.status + ')');
		const warnings = JSON.parse(response.headers.get('X-AutoDocs-Warnings') || '[]');
		const data = await response.json();
		if (version !== requestVersion) return;
		if (!Array.isArray(data) || !Array.isArray(warnings)) throw new Error('Unexpected directory response');
		allData = data; directoryIndex = model.index(data, directoryPrefs()); directoryReady = true;
		$('collection-warnings').hidden = !warnings.length;
		$('collection-warnings').textContent = warnings.length ? 'Discovery incomplete: ' + warnings.join('; ') : '';
		$('total-resources').textContent = data.length;
		$('total-namespaces').textContent = new Set(data.map(resource => resource.namespace)).size;
		$('total-hosts').textContent = directoryIndex.reduce((sum, item) => sum + item.endpoints.length, 0);
		filterOptions('namespace-filter', 'namespace', 'All namespaces'); filterOptions('kind-filter', 'kind', 'All kinds');
		$('loading').style.display = 'none'; $('content-container').style.display = 'block';
		performSearch($('search-input').value);
	} catch (error) {
		if (version !== requestVersion) return;
		$('loading').textContent = 'Error loading ' + (view === 'discovered' ? 'Discovered' : 'Applications') + ': ' + error.message;
		$('filtered-count').textContent = 'Results unavailable';
	}
}
function validName(name, exceptID) {
	if (!name.trim()) return 'Enter a group name.';
	if (directoryPrefs().groups.some(group => group.id !== exceptID && group.name === name.trim())) return 'A personal group already has that name.';
	return '';
}
function deleteGroup(id) {
	modifyDirectory(prefs => {
		prefs.groups = prefs.groups.filter(group => group.id !== id);
		prefs.assignments = Object.fromEntries(Object.entries(prefs.assignments).filter(([, group]) => group !== id));
		const deleted = model.groupID('custom', id);
		prefs.groupStars = prefs.groupStars.filter(group => group !== deleted);
		prefs.collapsed = prefs.collapsed.filter(group => group !== deleted);
	});
}
function renderEditor() {
	preserveFocus(() => {
		const list = $('editor-groups'); list.replaceChildren();
		const groups = directoryPrefs().groups;
		if (!groups.length) list.appendChild(element('p', 'muted', 'A fresh canvas. Create your first group above.'));
		groups.forEach((group, index) => {
			const row = element('div', 'editor-row');
			const label = element('label', '', 'Group ' + (index + 1));
			const input = control(element('input', 'text-input'), 'group-name-' + group.id);
			input.id = 'group-name-' + group.id; input.value = group.name; input.maxLength = 100; label.htmlFor = input.id;
			const actions = element('div', 'editor-actions');
			const save = control(button('Rename', () => {
				const error = validName(input.value, group.id);
				$('group-error').textContent = error;
				if (error) { input.focus(); return; }
				modifyDirectory(prefs => { prefs.groups.find(item => item.id === group.id).name = input.value.trim(); });
			}), 'rename-' + group.id);
			input.addEventListener('keydown', event => { if (event.key === 'Enter') { event.preventDefault(); save.click(); } });
			actions.appendChild(save);
			for (const [text, offset] of [['Move up', -1], ['Move down', 1]]) {
				const move = control(button(text, () => modifyDirectory(prefs => {
					const target = index + offset;
					[prefs.groups[index], prefs.groups[target]] = [prefs.groups[target], prefs.groups[index]];
				})), text + group.id);
				move.disabled = index + offset < 0 || index + offset >= groups.length; actions.appendChild(move);
			}
			actions.appendChild(button('Delete', () => { deleteGroup(group.id); $('new-group-name').focus(); }));
			row.append(label, input, actions); list.appendChild(row);
		});
	});
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
			event.preventDefault(); tabs[index].focus(); selectView(tabs[index].dataset.view);
		});
	}
	window.addEventListener('popstate', () => selectView(viewFromURL(), false));
	let searchTimeout;
	$('search-input').addEventListener('input', () => { clearTimeout(searchTimeout); searchTimeout = setTimeout(() => performSearch($('search-input').value), 140); });
	$('clear-search').addEventListener('click', () => { clearTimeout(searchTimeout); performSearch(''); $('search-input').focus(); });
	$('search-input').addEventListener('keydown', event => { if (event.key === 'Escape') { event.preventDefault(); performSearch(''); } });
	document.addEventListener('keydown', event => {
		if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k' && !event.target.closest('input, textarea, select, [contenteditable="true"]')) {
			event.preventDefault(); $('search-input').focus();
		}
	});
	for (const id of ['namespace-filter', 'kind-filter']) $(id).addEventListener('change', () => performSearch($('search-input').value));
	$('group-by').addEventListener('change', event => modifyDirectory(prefs => { prefs.groupBy = event.target.value; }));
	$('density').addEventListener('change', event => modifyDirectory(prefs => { prefs.density = event.target.value; }));
	$('favourites-toggle').addEventListener('click', () => { favouritesOnly = !favouritesOnly; syncControls(); performSearch($('search-input').value); });
	$('refresh-directory').addEventListener('click', loadDirectory);
	const editor = $('group-editor');
	$('edit-groups').addEventListener('click', () => { $('group-error').textContent = ''; renderEditor(); editor.showModal(); $('new-group-name').focus(); });
	$('close-editor').addEventListener('click', () => editor.close());
	editor.addEventListener('close', () => $('edit-groups').focus());
	$('new-group-form').addEventListener('submit', event => {
		event.preventDefault();
		const name = $('new-group-name').value.trim(), error = validName(name);
		$('group-error').textContent = error;
		if (error) return;
		const id = crypto.randomUUID ? crypto.randomUUID() : [...crypto.getRandomValues(new Uint8Array(16))].map(value => value.toString(16).padStart(2, '0')).join('');
		modifyDirectory(prefs => { prefs.groups.push({id, name}); prefs.groupBy = 'custom'; });
		$('new-group-name').value = ''; $('new-group-name').focus();
	});
	$('reset-layout').addEventListener('click', () => modifyDirectory(prefs => {
		const customIDs = new Set(prefs.groups.map(group => model.groupID('custom', group.id)));
		customIDs.add(model.groupID('custom'));
		const stars = prefs.groupStars.filter(id => !customIDs.has(id)), links = prefs.linkStars;
		Object.assign(prefs, AutoDocs.defaultDirectory(), {groupStars: stars, linkStars: links});
	}));
	$('clear-favourites').addEventListener('click', () => modifyDirectory(prefs => { prefs.groupStars = []; prefs.linkStars = []; }));
	AutoDocs.subscribe(() => {
		syncControls();
		if (directoryReady) { directoryIndex = model.index(allData, directoryPrefs()); performSearch($('search-input').value); }
		if (editor.open) renderEditor();
	});
	syncControls(); selectView(viewFromURL(), false);
});
