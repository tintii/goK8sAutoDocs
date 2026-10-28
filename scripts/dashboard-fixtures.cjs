/* Representative test data, never used by production pages. */
const resource = (name, namespace, group, kind = 'HTTPRoute', urls) => ({
	name, namespace, kind, labels: group === undefined ? {} : {'k8sautodocs.io/group': group},
	endpoints: (urls || ['https://' + name + '.homelab.test']).map(url => ({url, gateway: {name: 'homelab', namespace: 'edge'}, listener: 'https'})),
	relatedResources: [{name: name + '-service', namespace, kind: 'Service', labels: {app: name}}]
});
const published = [
	resource('argocd', 'argocd', 'Infrastructure'),
	resource('grafana', 'monitoring', 'Infrastructure'),
	resource('rancher', 'cattle-system', 'Infrastructure'),
	resource('sonarr', 'media', 'Media'),
	resource('radarr', 'media', 'Media'),
	resource('jellyfin', 'media', 'Media'),
	resource('home-assistant', 'home', 'Home'),
	resource('paperless', 'productivity', 'Home'),
	{...resource('backup-console-with-a-long-resource-name', 'backups', undefined, 'ingress', []), issues: ['No concrete hostname is available.']}
];
const discovered = [resource('unpublished-service', 'default', 'Infrastructure'), {...resource('pending-route', 'dev', undefined, 'HTTPRoute', []), issues: ['Parent Gateway is unavailable.']}];
const versions = [
	{name: 'argocd-server', namespace: 'argocd', kind: 'Deployment', containers: [{name: 'server', type: 'main', image: 'quay.io/argoproj/argocd', tag: 'fixture', sha: 'sha256:abcd1234'}]},
	{name: 'grafana', namespace: 'monitoring', kind: 'Deployment', containers: [{name: 'grafana', type: 'main', image: 'grafana/grafana', tag: 'fixture', sha: ''}, {name: 'init', type: 'init', image: 'busybox', tag: 'fixture', sha: ''}]}
];
function setup(data = published, discovery = discovered, extra = '') {
	return 'window.fixtureData = ' + JSON.stringify(data) + '; window.fixtureDiscovered = ' + JSON.stringify(discovery) + '; window.fixtureVersions = ' + JSON.stringify(versions) + ';' +
		'window.requestLog = []; window.fetch = async (url, init = {}) => { requestLog.push({url, method:init.method || "GET"}); return {ok:true, headers:{get:()=> "[]"}, json: async()=>url.includes("cluster-info") ? {clusterName:"homelab / k3s"} : url.endsWith("/home") ? {title:"Cluster overview",environment:"Homelab",description:"Application links, workload status and container versions.",motd:"",links:[],footer:"AutoDocs"} : url.endsWith("/service-status") ? {ready:0,attention:0,inactive:0,workloads:[],checkedAt:new Date().toISOString()} : url.includes("versions-dashboard") ? fixtureVersions : url.includes("view=discovered") ? fixtureDiscovered : fixtureData}; };' + extra;
}
module.exports = {resource, published, discovered, versions, setup};
