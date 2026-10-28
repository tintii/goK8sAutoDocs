/* Personal table controls operate on the loaded inventory; API values remain text. */
document.addEventListener('DOMContentLoaded', () => {
	const {element} = AutoDocs;
	const $ = id => document.getElementById(id);
	const columns = [
		{key: 'resource', label: 'Resource', className: '', floor: 150, cap: 340, flexible: true},
		{key: 'namespace', label: 'Namespace', className: 'mono', floor: 130, cap: 240},
		{key: 'kind', label: 'Kind', className: 'resource-kind', floor: 100, cap: 140},
		{key: 'container', label: 'Container', className: '', floor: 140, cap: 280, flexible: true},
		{key: 'type', label: 'Type', className: 'container-type', floor: 80, cap: 100},
		{key: 'image', label: 'Image', className: 'image-name', floor: 180, cap: 400, flexible: true},
		{key: 'tag', label: 'Tag', className: 'image-tag', floor: 100, cap: 220, flexible: true},
		{key: 'sha', label: 'SHA', className: 'mono', floor: 160, cap: 250, flexible: true},
		{key: 'cluster', label: 'Cluster', className: 'mono', floor: 140, cap: 260},
		{key: 'environment', label: 'Environment', className: '', floor: 120, cap: 180},
		{key: 'datacentre', label: 'Datacentre', className: '', floor: 110, cap: 180},
	];
	const collator = new Intl.Collator(undefined, {numeric: true, sensitivity: 'base'});
	const table = $('versions-table');
	const region = $('table-container');
	const measure = document.createElement('canvas').getContext('2d');
	let data = [], rows = [], widths = [], preferred = [], minimum = [], floors = [];
	let preferences = AutoDocs.preferences.versions;
	let loaded = false;
	let layoutWidth = 0;
	let drag = null;
	const text = value => value == null ? '' : String(value);
	const clamp = (value, min, max = 1200) => Math.max(min, Math.min(max, Math.round(value)));

	function changeSort(key) {
		AutoDocs.update(prefs => {
			const sort = prefs.versions.sort;
			const index = sort.findIndex(item => item.key === key);
			if (index >= 0) {
				if (sort[index].direction === 'asc') sort[index].direction = 'desc';
				else sort.splice(index, 1);
			} else {
				if (sort.length === 2) sort.pop();
				sort.push({key, direction: 'asc'});
			}
		});
	}
	function updateHeadings() {
		for (const column of columns) {
			const index = preferences.sort.findIndex(item => item.key === column.key);
			const active = preferences.sort[index];
			column.heading.removeAttribute('aria-sort');
			if (index === 0) column.heading.setAttribute('aria-sort', active.direction === 'asc' ? 'ascending' : 'descending');
			column.indicator.textContent = active ? (active.direction === 'asc' ? '↑' : '↓') + (index + 1) : '↕';
			column.sortButton.dataset.active = String(Boolean(active));
			const next = active ? (active.direction === 'asc' ? 'Sort descending.' : 'Remove from sorting.') :
				(preferences.sort.length ? 'Add as secondary sort ascending.' : 'Sort ascending.');
			column.sortButton.setAttribute('aria-label', column.label + '. ' + (active ? 'Sort ' + (index + 1) + ', ' + (active.direction === 'asc' ? 'ascending' : 'descending') + '. ' : '') + next);
		}
		$('sort-summary').textContent = preferences.sort.length ? 'Sorted by ' + preferences.sort.map((item, index) =>
			(index + 1) + '. ' + columns.find(column => column.key === item.key).label + ' ' + (item.direction === 'asc' ? '↑ ascending' : '↓ descending')).join(', then ') : 'Original order';
		$('clear-sort').disabled = !preferences.sort.length;
	}
	function compare(a, b) {
		for (const {key, direction} of preferences.sort) {
			const left = a.values[key], right = b.values[key];
			if (Boolean(left) !== Boolean(right)) return left ? -1 : 1;
			const order = collator.compare(left, right);
			if (order) return direction === 'asc' ? order : -order;
		}
		return a.index - b.index;
	}
	function renderRows() {
		const query = $('search-input').value.trim().toLowerCase();
		const filtered = new Set(data.filter(resource => Inventory.matches(resource) && [resource.name, resource.namespace, resource.kind, resource.clusterId, resource.clusterName, ...Object.values(resource.classification || {}).flatMap(value => [value.id, value.display]),
			...(resource.containers || []).flatMap(container => [container.name, container.image, container.tag, container.sha])]
			.some(value => text(value).toLowerCase().includes(query))));
		const scoped = data.filter(resource => Inventory.matches(resource));
		$('total-resources').textContent = scoped.length;
		$('total-containers').textContent = scoped.reduce((sum, resource) => sum + (resource.containers || []).length, 0);
		$('unique-images').textContent = new Set(scoped.flatMap(resource => (resource.containers || []).map(container => container.image + ':' + container.tag))).size;
		const visible = rows.filter(row => filtered.has(row.resource)).sort(compare);
		const body = $('table-body');
		const fragment = document.createDocumentFragment();
		for (const entry of visible) {
			const row = element('tr');
			for (const column of columns) {
				const value = entry.values[column.key] || '—';
				const cell = element('td');
				const content = element('span', column.className, value);
				content.title = value;
				cell.appendChild(content); row.appendChild(cell);
			}
			fragment.appendChild(row);
		}
		if (!visible.length) {
			const row = element('tr'), cell = element('td', 'muted', data.length ? 'No matching resources.' : 'No container versions found.');
			cell.colSpan = columns.length; row.appendChild(cell); fragment.appendChild(row);
		}
		body.replaceChildren(fragment);
		$('filtered-count').textContent = 'Showing ' + filtered.size + ' of ' + scoped.length + ' resources';
		$('clear-search').style.display = query ? 'inline-flex' : 'none';
	}
	function measureColumn(column, index) {
		const sample = table.querySelector('tbody tr td:nth-child(' + (index + 1) + ') span');
		const sampleStyle = getComputedStyle(sample || column.sortButton);
		measure.font = sampleStyle.font;
		let size = 0;
		for (const row of rows) size = Math.max(size, measure.measureText(row.values[column.key] || '—').width);
		const badge = ['kind', 'type', 'tag'].includes(column.key) ? 12 : 0;
		return clamp(size + 28 + badge, minimum[index]);
	}
	function measureDefaults() {
		minimum = columns.map(column => {
			measure.font = getComputedStyle(column.sortButton).font;
			return Math.max(64, Math.ceil(measure.measureText(column.label).width) + 64);
		});
		preferred = columns.map((column, index) => Math.min(column.cap, measureColumn(column, index)));
		floors = columns.map((column, index) => Math.min(preferred[index], Math.max(minimum[index], column.floor)));
	}
	function applyWidths() {
		widths.forEach((width, index) => {
			columns[index].col.style.width = width + 'px';
			columns[index].handle.setAttribute('aria-valuenow', width);
			columns[index].handle.setAttribute('aria-valuemin', minimum[index] || 64);
		});
		table.style.width = widths.reduce((sum, width) => sum + width, 0) + 'px';
	}
	function layout() {
		if (!loaded || drag) return;
		widths = columns.map((column, index) => clamp(preferences.widths[column.key] ?? preferred[index], minimum[index]));
		const automatic = columns.map((column, index) => index).filter(index => !Object.hasOwn(preferences.widths, columns[index].key) && columns[index].flexible);
		const total = widths.reduce((sum, width) => sum + width, 0);
		const available = region.clientWidth;
		layoutWidth = available;
		if (total > available) {
			const room = automatic.reduce((sum, index) => sum + Math.max(0, widths[index] - floors[index]), 0);
			const shrink = Math.min(total - available, room);
			if (room) for (const index of automatic) widths[index] -= Math.floor(shrink * (widths[index] - floors[index]) / room);
		} else if (automatic.length) {
			const extra = Math.floor((available - total) / automatic.length);
			for (const index of automatic) widths[index] = Math.min(1200, widths[index] + extra);
		}
		applyWidths();
	}
	function saveWidth(index, width) {
		AutoDocs.update(prefs => { prefs.versions.widths[columns[index].key] = clamp(width, minimum[index]); });
	}
	function fitColumn(index) {
		if (loaded) saveWidth(index, measureColumn(columns[index], index));
	}
	function finishDrag(cancel) {
		if (!drag) return;
		const current = drag; drag = null;
		if (cancel) { widths = current.before; applyWidths(); }
		document.body.classList.remove('resizing-column');
		if (current.handle.hasPointerCapture(current.pointer)) current.handle.releasePointerCapture(current.pointer);
		if (!cancel) saveWidth(current.index, widths[current.index]);
	}
	columns.forEach((column, index) => {
		column.col = element('col'); $('versions-columns').appendChild(column.col);
		column.heading = element('th'); column.heading.scope = 'col'; column.heading.dataset.column = column.key;
		column.sortButton = element('button', 'column-sort'); column.sortButton.type = 'button'; column.sortButton.dataset.sort = column.key;
		column.indicator = element('span', 'sort-indicator'); column.indicator.setAttribute('aria-hidden', 'true');
		column.sortButton.append(element('span', '', column.label), column.indicator);
		column.sortButton.addEventListener('click', () => changeSort(column.key));
		column.handle = element('span', 'column-resize'); column.handle.tabIndex = 0;
		for (const [key, value] of Object.entries({role: 'separator', 'aria-orientation': 'vertical', 'aria-label': 'Resize ' + column.label + ' column',
			'aria-valuemin': 64, 'aria-valuemax': 1200, 'aria-valuenow': 100, 'aria-describedby': 'resize-help'})) column.handle.setAttribute(key, value);
		column.handle.title = 'Drag to resize; double-click to fit. Arrow keys resize; Shift for larger steps; Home to fit.';
		column.handle.addEventListener('pointerdown', event => {
			if (!loaded || event.button !== 0 || drag) return;
			event.preventDefault(); column.handle.focus({preventScroll: true});
			drag = {index, handle: column.handle, pointer: event.pointerId, x: event.clientX, before: [...widths]};
			column.handle.setPointerCapture(event.pointerId); document.body.classList.add('resizing-column');
		});
		column.handle.addEventListener('pointermove', event => {
			if (!drag || event.pointerId !== drag.pointer) return;
			widths[index] = clamp(drag.before[index] + event.clientX - drag.x, minimum[index]); applyWidths();
		});
		column.handle.addEventListener('pointerup', event => { if (drag?.pointer === event.pointerId) finishDrag(false); });
		column.handle.addEventListener('pointercancel', () => finishDrag(true));
		column.handle.addEventListener('lostpointercapture', () => finishDrag(true));
		column.handle.addEventListener('dblclick', () => fitColumn(index));
		column.handle.addEventListener('keydown', event => {
			if (!loaded || drag) return;
			if (event.key === 'Home') { event.preventDefault(); fitColumn(index); }
			if (['ArrowLeft', 'ArrowRight'].includes(event.key)) {
				event.preventDefault(); saveWidth(index, widths[index] + (event.key === 'ArrowRight' ? 1 : -1) * (event.shiftKey ? 50 : 10));
			}
		});
		column.heading.append(column.sortButton, column.handle); $('versions-head').appendChild(column.heading);
	});
	$('fit-columns').addEventListener('click', () => {
		finishDrag(true);
		if (loaded) measureDefaults();
		AutoDocs.update(prefs => { prefs.versions.widths = {}; });
		layout();
	});
	$('clear-sort').addEventListener('click', () => AutoDocs.update(prefs => { prefs.versions.sort = []; }));
	AutoDocs.subscribe(() => {
		const next = AutoDocs.preferences.versions;
		if (JSON.stringify(next) === JSON.stringify(preferences)) return;
		finishDrag(true);
		const sortChanged = JSON.stringify(next.sort) !== JSON.stringify(preferences.sort);
		const removedWidth = Object.keys(preferences.widths).some(key => !Object.hasOwn(next.widths, key));
		preferences = next; updateHeadings();
		if (loaded) {
			if (sortChanged) renderRows();
			if (removedWidth) layout();
			else {
				for (const [index, column] of columns.entries()) {
					if (Object.hasOwn(preferences.widths, column.key)) widths[index] = clamp(preferences.widths[column.key], minimum[index]);
				}
				applyWidths();
			}
		}
	});
	new ResizeObserver(() => { if (region.clientWidth !== layoutWidth) layout(); }).observe(region);
	updateHeadings();
	let timer;
	$('search-input').addEventListener('input', () => { clearTimeout(timer); timer = setTimeout(renderRows, 140); });
	const clear = () => { $('search-input').value = ''; renderRows(); $('search-input').focus(); };
	$('clear-search').addEventListener('click', clear);
	$('search-input').addEventListener('keydown', event => { if (event.key === 'Escape') { event.preventDefault(); clear(); } });
	document.addEventListener('keydown', event => {
		if (event.key === 'Escape' && drag) { event.preventDefault(); finishDrag(true); }
		if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k' && !event.target.closest('input, textarea, select, [contenteditable="true"]')) {
			event.preventDefault(); $('search-input').focus();
		}
	});
	function loadVersions() { return fetch('/api/v1/versions-dashboard', {cache:'no-store'}).then(response => {
		if (!response.ok) throw new Error('Request failed (' + response.status + ')'); return response.json();
	}).then(response => {
		// Retain compatibility with older servers that return null for an empty inventory.
		if (response === null) response = [];
		if (!Array.isArray(response)) throw new Error('Unexpected versions response');
		data = response;
		rows = data.flatMap(resource => (resource.containers || []).map(container => ({resource, values: {
			cluster: text(resource.clusterName || resource.clusterId), environment: text(resource.classification?.environment?.display), datacentre: text(resource.classification?.datacentre?.display),
			resource: text(resource.name), namespace: text(resource.namespace), kind: text(resource.kind),
			container: text(container.name), type: container.type === 'init' ? 'Init' : 'Main', image: text(container.image), tag: text(container.tag), sha: text(container.sha),
		}})));
		rows.forEach((row, index) => { row.index = index; });
		$('total-resources').textContent = data.length;
		$('total-containers').textContent = rows.length;
		$('unique-images').textContent = new Set(rows.map(row => row.values.image + ':' + row.values.tag)).size;
		$('loading').style.display = 'none'; region.style.display = 'block';
		renderRows(); loaded = true; measureDefaults(); layout();
	}).catch(error => { $('loading').textContent = 'Error loading data: ' + error.message; $('filtered-count').textContent = 'Results unavailable'; }); }
	window.addEventListener('inventory-generation', loadVersions);
	window.addEventListener('inventory-scope', renderRows);
	loadVersions();
});
