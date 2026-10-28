/* Versions retain resource-level filtering; render API values as text. */
document.addEventListener('DOMContentLoaded', () => {
	const {element} = AutoDocs;
	const $ = id => document.getElementById(id);
	let data = [];
	function search() {
		const query = $('search-input').value.trim().toLowerCase();
		const filtered = data.filter(resource => [resource.name, resource.namespace, resource.kind,
			...(resource.containers || []).flatMap(container => [container.name, container.image, container.tag, container.sha])]
			.some(value => String(value || '').toLowerCase().includes(query)));
		const body = $('table-body'); body.replaceChildren();
		for (const resource of filtered) for (const container of resource.containers || []) {
			const row = element('tr');
			for (const [value, className] of [[resource.name, ''], [resource.namespace, 'mono'], [resource.kind, 'resource-kind'], [container.name, ''],
				[container.type === 'init' ? 'Init' : 'Main', 'container-type'], [container.image, 'image-name'], [container.tag, 'image-tag'], [container.sha || '—', 'mono']]) {
				const cell = element('td'); cell.appendChild(element('span', className, value)); row.appendChild(cell);
			}
			body.appendChild(row);
		}
		if (!body.childElementCount) { const row = element('tr'), cell = element('td', 'muted', data.length ? 'No matching resources.' : 'No container versions found.'); cell.colSpan = 8; row.appendChild(cell); body.appendChild(row); }
		$('filtered-count').textContent = 'Showing ' + filtered.length + ' of ' + data.length + ' resources';
		$('clear-search').style.display = query ? 'inline-flex' : 'none';
	}
	let timer;
	$('search-input').addEventListener('input', () => { clearTimeout(timer); timer = setTimeout(search, 140); });
	const clear = () => { $('search-input').value = ''; search(); $('search-input').focus(); };
	$('clear-search').addEventListener('click', clear);
	$('search-input').addEventListener('keydown', event => { if (event.key === 'Escape') { event.preventDefault(); clear(); } });
	document.addEventListener('keydown', event => {
		if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k' && !event.target.closest('input, textarea, select, [contenteditable="true"]')) {
			event.preventDefault(); $('search-input').focus();
		}
	});
	fetch('/api/v1/versions-dashboard').then(response => {
		if (!response.ok) throw new Error('Request failed (' + response.status + ')'); return response.json();
	}).then(response => {
		// The existing Go handler serializes an empty nil inventory as JSON null.
		if (response === null) response = [];
		if (!Array.isArray(response)) throw new Error('Unexpected versions response');
		data = response;
		const containers = data.flatMap(resource => resource.containers || []);
		$('total-resources').textContent = data.length;
		$('total-containers').textContent = containers.length;
		$('unique-images').textContent = new Set(containers.map(container => container.image + ':' + container.tag)).size;
		$('loading').style.display = 'none'; $('table-container').style.display = 'block'; search();
	}).catch(error => { $('loading').textContent = 'Error loading data: ' + error.message; $('filtered-count').textContent = 'Results unavailable'; });
});
