/* Pure directory projection; persisted identities never supply destinations. */
(() => {
	'use strict';
	const groupLabel = 'k8sautodocs.io/group';
	const compare = (a, b) => a < b ? -1 : a > b ? 1 : 0;
	const sourceTuple = resource => resource.clusterId ? [resource.clusterId, resource.kind, resource.namespace, resource.name] : [resource.kind, resource.namespace, resource.name];
	const sourceID = resource => JSON.stringify(sourceTuple(resource));
	const linkID = (resource, url) => JSON.stringify(['link', sourceTuple(resource), url]);
	const groupID = (mode, value) => JSON.stringify(value === undefined ? ['group', mode + '-fallback'] : ['group', mode, value]);
	function group(resource, mode, prefs) {
		let value, name, fallback = false;
		if (mode === 'label') { value = resource.labels?.[groupLabel] || undefined; name = value || 'Ungrouped'; fallback = value === undefined; }
		if (mode === 'namespace' || mode === 'kind') { value = resource[mode]; name = value; if (mode === 'namespace' && resource.clusterId) { value = [resource.clusterId, resource.namespace]; name = (resource.clusterName || resource.clusterId) + ' / ' + resource.namespace; } }
		if (mode === 'cluster') { value = resource.clusterId; name = resource.clusterName || value || 'Unknown cluster'; fallback = !value; }
		if (mode === 'environment' || mode === 'datacentre') { const dimension = resource.classification?.[mode]; value = dimension?.id; name = dimension?.display || (mode === 'environment' ? 'Unclassified environment' : 'Unspecified datacentre'); fallback = !dimension; }
		if (mode === 'environment-cluster' || mode === 'cluster-environment') {
			const environment = resource.classification?.environment;
			value = mode === 'environment-cluster' ? [environment?.id ?? null, resource.clusterId ?? null] : [resource.clusterId ?? null, environment?.id ?? null];
			const parts = [environment?.display || 'Unclassified environment', resource.clusterName || resource.clusterId || 'Unknown cluster'];
			name = (mode === 'environment-cluster' ? parts : parts.reverse()).join(' / '); fallback = !environment;
		}
		if (mode === 'custom') {
			const custom = prefs.groups.find(item => item.id === prefs.assignments[sourceID(resource)]);
			value = custom?.id; name = custom?.name || 'Unassigned'; fallback = !custom;
		}
		return {id: groupID(mode, value), name, mode, fallback, path: Array.isArray(value) ? value : undefined};
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
		return [...data].sort((a, b) => compare(a.clusterId || '', b.clusterId || '') || compare(a.namespace, b.namespace) || compare(a.kind, b.kind) || compare(a.name, b.name)).map(resource => {
			const memberships = ['label', 'namespace', 'kind', 'custom', 'cluster', 'environment', 'datacentre', 'environment-cluster', 'cluster-environment'].map(mode => group(resource, mode, prefs));
			const fields = [resource.name, resource.namespace, resource.kind, resource['ingress-class'], resource.labels?.[groupLabel],
				resource.clusterId, resource.clusterName, ...Object.values(resource.classification || {}).flatMap(value => [value.id, value.display]), memberships.find(item => item.mode === 'custom').name,
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
			if (window.Inventory && !Inventory.matches(item.resource)) continue;
			if (filters.namespace && item.resource.namespace !== filters.namespace || filters.kind && item.resource.kind !== filters.kind) continue;
			if (!tokens.every(token => item.fields.some(field => field.includes(token)))) continue;
			const wholeGroup = item.memberships.some(member => groupStars.has(member.id));
			const links = filters.favourites && !wholeGroup ? item.endpoints.filter(endpoint => linkStars.has(linkID(item.resource, endpoint.url))) : item.endpoints;
			if (filters.favourites && !wholeGroup && !links.length) continue;
			rows.push({...item, endpoints: links});
		}
		const mode = prefs.groupBy;
		if (mode === 'none') return {rows, sections: [{id: 'flat', name: '', rows}]};
		const sections = new Map();
		for (const row of rows) {
			const membership = row.memberships.find(member => member.mode === mode);
			if (!sections.has(membership.id)) sections.set(membership.id, {...membership, rows: []});
			sections.get(membership.id).rows.push(row);
		}
		return {rows, sections: [...sections.values()].sort((a, b) => {
			if (mode === 'environment-cluster' || mode === 'cluster-environment') {
				for (let index = 0; index < 2; index++) {
					if ((a.path[index] === null) !== (b.path[index] === null)) return a.path[index] === null ? 1 : -1;
					const order = compare(a.path[index] || '', b.path[index] || '');
					if (order) return order;
				}
			}
			if (a.fallback !== b.fallback) return a.fallback ? 1 : -1;
			if (mode === 'custom') return prefs.groups.findIndex(item => groupID('custom', item.id) === a.id) - prefs.groups.findIndex(item => groupID('custom', item.id) === b.id);
			return compare(a.name, b.name);
		})};
	}
	window.DirectoryModel = {groupLabel, sourceID, linkID, groupID, group, endpoints, index, project};
})();
