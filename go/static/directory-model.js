/* Pure directory projection; persisted identities never supply destinations. */
(() => {
	'use strict';
	const groupLabel = 'k8sautodocs.io/group';
	const compare = (a, b) => a < b ? -1 : a > b ? 1 : 0;
	const sourceTuple = resource => [resource.kind, resource.namespace, resource.name];
	const sourceID = resource => JSON.stringify(sourceTuple(resource));
	const linkID = (resource, url) => JSON.stringify(['link', sourceTuple(resource), url]);
	const groupID = (mode, value) => JSON.stringify(value === undefined ? ['group', mode + '-fallback'] : ['group', mode, value]);
	function group(resource, mode, prefs) {
		let value, name, fallback = false;
		if (mode === 'label') { value = resource.labels?.[groupLabel] || undefined; name = value || 'Ungrouped'; fallback = value === undefined; }
		if (mode === 'namespace' || mode === 'kind') { value = resource[mode]; name = value; }
		if (mode === 'custom') {
			const custom = prefs.groups.find(item => item.id === prefs.assignments[sourceID(resource)]);
			value = custom?.id; name = custom?.name || 'Unassigned'; fallback = !custom;
		}
		return {id: groupID(mode, value), name, mode, fallback};
	}
	function endpoints(resource) {
		const urls = new Map();
		for (const endpoint of resource.endpoints || []) {
			// Only the API's concrete web URLs are eligible, never saved strings.
			try {
				const parsed = new URL(endpoint.url);
				if (!['http:', 'https:'].includes(parsed.protocol)) continue;
			} catch { continue; }
			if (!urls.has(endpoint.url)) urls.set(endpoint.url, {...endpoint, attachments: []});
			if (endpoint.gateway) urls.get(endpoint.url).attachments.push({gateway: endpoint.gateway, listener: endpoint.listener});
		}
		return [...urls.values()];
	}
	function index(data, prefs) {
		return [...data].sort((a, b) => compare(a.namespace, b.namespace) || compare(a.kind, b.kind) || compare(a.name, b.name)).map(resource => {
			const memberships = ['label', 'namespace', 'kind', 'custom'].map(mode => group(resource, mode, prefs));
			const fields = [resource.name, resource.namespace, resource.kind, resource['ingress-class'], resource.labels?.[groupLabel],
				memberships.find(item => item.mode === 'custom').name,
				...(resource.hostnames || []).map(host => host.host),
				...(resource.endpoints || []).flatMap(item => [item.url, item.listener, item.gateway?.name, item.gateway?.namespace]),
				...(resource.relatedResources || []).flatMap(item => [item.name, item.namespace, item.kind, ...Object.values(item.labels || {})])];
			return {resource, source: sourceID(resource), endpoints: endpoints(resource), memberships, fields: fields.filter(value => value !== undefined && value !== null).map(value => String(value).toLowerCase())};
		});
	}
	function project(indexed, prefs, filters) {
		const tokens = filters.query.trim().toLowerCase().split(/\s+/).filter(Boolean);
		const groupStars = new Set(prefs.groupStars), linkStars = new Set(prefs.linkStars);
		const rows = [];
		for (const item of indexed) {
			if (filters.namespace && item.resource.namespace !== filters.namespace || filters.kind && item.resource.kind !== filters.kind) continue;
			if (!tokens.every(token => item.fields.some(field => field.includes(token)))) continue;
			const wholeGroup = filters.view === 'published' && item.memberships.some(member => groupStars.has(member.id));
			const links = filters.favourites && !wholeGroup ? item.endpoints.filter(endpoint => linkStars.has(linkID(item.resource, endpoint.url))) : item.endpoints;
			if (filters.favourites && !wholeGroup && !links.length) continue;
			rows.push({...item, endpoints: links});
		}
		const mode = filters.view === 'discovered' ? 'none' : prefs.groupBy;
		if (mode === 'none') return {rows, sections: [{id: 'flat', name: '', rows}]};
		const sections = new Map();
		for (const row of rows) {
			const membership = row.memberships.find(member => member.mode === mode);
			if (!sections.has(membership.id)) sections.set(membership.id, {...membership, rows: []});
			sections.get(membership.id).rows.push(row);
		}
		return {rows, sections: [...sections.values()].sort((a, b) => {
			if (a.fallback !== b.fallback) return a.fallback ? 1 : -1;
			if (mode === 'custom') return prefs.groups.findIndex(item => groupID('custom', item.id) === a.id) - prefs.groups.findIndex(item => groupID('custom', item.id) === b.id);
			return compare(a.name, b.name);
		})};
	}
	window.DirectoryModel = {groupLabel, sourceID, linkID, groupID, group, endpoints, index, project};
})();
