/* Pure directory projection; persisted identities never supply destinations. */
(() => {
	'use strict';
	const groupLabel = 'k8sautodocs.io/group';
	const compare = (a, b) => a < b ? -1 : a > b ? 1 : 0;
	const manual = resource => resource.sourceType === 'manual-link';
	const kindLabel = resource => manual(resource) ? 'Manual link' : resource.kind;
	const sourceTuple = resource => manual(resource) ? ['manual-link', resource.provenance.clusterId, resource.provenance.namespace, resource.provenance.configMapName, resource.provenance.entryId] : resource.clusterId ? [resource.clusterId, resource.kind, resource.namespace, resource.name] : [resource.kind, resource.namespace, resource.name];
	const sourceID = resource => JSON.stringify(sourceTuple(resource));
	const linkID = (resource, url) => JSON.stringify(['link', sourceTuple(resource), url]);
	const groupID = (mode, value) => JSON.stringify(value === undefined ? ['group', mode + '-fallback'] : ['group', mode, value]);
	function group(resource, mode, prefs) {
		let value, name, fallback = false;
		// Shared manual components have typed identities; discovered identities
		// remain compatible with existing browser preferences.
		if (manual(resource) && ['cluster','namespace','environment','datacentre','environment-cluster','cluster-environment'].includes(mode)) {
			const component = dimension => {
				const id = dimension === 'cluster' ? resource.clusterId : dimension === 'namespace' ? resource.namespace : resource.classification?.[dimension]?.id;
				const display = dimension === 'cluster' ? resource.clusterName || id : dimension === 'namespace' ? id : resource.classification?.[dimension]?.display;
				return id === undefined ? {value:{shared:dimension}, name:'Shared across ' + ({cluster:'clusters',namespace:'namespaces',environment:'environments',datacentre:'datacentres'})[dimension], rank:1} : {value:id,name:display,rank:0};
			};
			const dimensions = mode === 'namespace' ? ['cluster','namespace'] : mode.includes('-') ? mode.split('-') : [mode];
			const parts = dimensions.map(component);
			value = parts.length === 1 ? parts[0].value : parts.map(part => part.value);
			return {id:groupID(mode,value),name:parts.map(part=>part.name).join(' / '),mode,fallback:false,sortPath:parts.map(part=>[part.rank,parts.length === 1 || mode === 'namespace' ? part.name : part.value])};
		}
		if (mode === 'label') { value = resource.labels?.[groupLabel] || undefined; name = value || 'Ungrouped'; fallback = value === undefined; }
		if (mode === 'namespace' || mode === 'kind') { value = resource[mode]; name = mode === 'kind' ? kindLabel(resource) : value; if (mode === 'namespace' && resource.clusterId) { value = [resource.clusterId, resource.namespace]; name = (resource.clusterName || resource.clusterId) + ' / ' + resource.namespace; } }
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
		let sortPath;
		if (['cluster','environment','datacentre'].includes(mode)) sortPath=[[fallback ? 2 : 0,name]];
		if (mode === 'namespace') sortPath=[[0,resource.clusterName || resource.clusterId || ''],[0,resource.namespace || '']];
		if (mode === 'environment-cluster' || mode === 'cluster-environment') sortPath=value.map(part=>[part === null ? 2 : 0,part]);
		return {id: groupID(mode, value), name, mode, fallback, path: Array.isArray(value) ? value : undefined,sortPath};
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
		return [...data].sort((a, b) => compare(a.clusterId || '', b.clusterId || '') || compare(a.namespace || '', b.namespace || '') || compare(a.kind, b.kind) || compare(a.name, b.name) || compare(sourceID(a), sourceID(b))).map(resource => {
			const memberships = ['label', 'namespace', 'kind', 'custom', 'cluster', 'environment', 'datacentre', 'environment-cluster', 'cluster-environment'].map(mode => group(resource, mode, prefs));
			const fields = [resource.title, kindLabel(resource), resource.name, resource.namespace, resource.kind, resource['ingress-class'], resource.labels?.[groupLabel],
				resource.serviceType, resource.externalName, ...(resource.dns?.targets || []),
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
			if (!filters.unscoped && window.Inventory && !Inventory.matches(item.resource, {...Inventory.scope,namespace:filters.namespace})) continue;
			if (filters.namespace && !(manual(item.resource) && !item.resource.namespace) && item.resource.namespace !== filters.namespace || filters.kind && item.resource.kind !== filters.kind) continue;
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
			if (a.sortPath && b.sortPath) {
				const left=a.sortPath, right=b.sortPath;
				for (let i=0;i<Math.max(left.length,right.length);i++) {
					const x=left[i] || [0,''], y=right[i] || [0,''];
					const order=x[0]-y[0] || compare(typeof x[1] === 'string' ? x[1] : '',typeof y[1] === 'string' ? y[1] : '');
					if(order) return order;
				}
			}

			if (a.fallback !== b.fallback) return a.fallback ? 1 : -1;
			if (mode === 'label' && filters.view === 'published') {
				const rank = group => group.rows[0].resource.directoryGroupRank || Infinity;
				return compare(rank(a), rank(b)) || compare(a.name, b.name);
			}
			if (mode === 'custom') return prefs.groups.findIndex(item => groupID('custom', item.id) === a.id) - prefs.groups.findIndex(item => groupID('custom', item.id) === b.id);
			return compare(a.name, b.name);
		})};
	}
	function featured(rows) {
		return rows.flatMap(({resource, endpoints}) => endpoints
			.filter(({url}) => Object.hasOwn(resource.featuredLinkRanks || {}, url))
			.map(({url}) => ({resource, url, rank: resource.featuredLinkRanks[url]})))
			.sort((a, b) => a.rank - b.rank);
	}
	function quickResults(rows, limit) {
		const seen = new Set();
		const items = [];
		for (const {resource, source, endpoints} of rows) {
			for (const endpoint of endpoints.length ? endpoints : [null]) {
				const url = endpoint?.url;
				const id = url ? linkID(resource, url) : source;
				if (seen.has(id)) continue;
				seen.add(id);
				if (items.length < limit) items.push({id, resource, source, url});
			}
		}
		return {total: seen.size, items};
	}
	window.DirectoryModel = {manual, kindLabel, groupLabel, sourceID, linkID, groupID, group, endpoints, index, project, featured, quickResults};
})();
