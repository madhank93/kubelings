// src/data/further-reading.ts
// Cited real-world incidents that are NOT (yet) reproduced as Kubelings lessons —
// talks, postmortems and blog write-ups worth reading. Rendered as the "Further
// reading" appendix on /catalog. Hand-maintained (source: the old Incident Library).

export type Reading = {
  company: string;
  title: string;
  url: string;
  module?: string;   // rough curriculum area it maps to
  caseStudy?: string; // local /incidents/* write-up, if one exists
};

export const FURTHER_READING: Reading[] = [
  { company: 'Tinder',        title: 'Move to Kubernetes at scale — 250k rps DNS, conntrack & ARP cache outage', url: 'https://medium.com/tinder/tinders-move-to-kubernetes-cda2a6372f44', module: 'M4', caseStudy: '/incidents/tinder-scale-migration/' },
  { company: 'CircleCI',      title: 'kubelet/kube-proxy version skew corrupted iptables mid-upgrade', url: 'https://discuss.circleci.com/t/incident-report-2023-03-14-delays-starting-jobs/47555', module: 'M8', caseStudy: '/incidents/circleci-version-skew/' },
  { company: 'Heroku',        title: 'Unattended system update flushed network routes fleet-wide', url: 'https://www.heroku.com/blog/summary-of-june-10-outage/', module: 'M8', caseStudy: '/incidents/heroku-host-update/' },
  { company: 'Monzo',         title: 'Anatomy of a Production Kubernetes Outage (KubeCon talk)', url: 'https://www.youtube.com/watch?v=OUYTNywPk-s', module: 'M9' },
  { company: 'Datadog',       title: '10 ways to shoot yourself in the foot with Kubernetes', url: 'https://www.youtube.com/watch?v=QKI-JRs2RIE', module: 'M4' },
  { company: 'Zalando',       title: 'Kubelet --kube-api-qps starves CD platform builds', url: 'https://github.com/zalando-incubator/kubernetes-on-aws/blob/dev/docs/postmortems/jun-2019-kubelet-qps.md', module: 'M7' },
  { company: 'Zalando',       title: 'A million ways to crash your cluster (DevOpsCon)', url: 'https://www.slideshare.net/try_except_/running-kubernetes-in-production-a-million-ways-to-crash-your-cluster-devopscon-munich-2018', module: 'M8' },
  { company: 'Zalando',       title: "Let's talk about failures (NotReady nodes, ELB, CoreDNS)", url: 'https://www.slideshare.net/try_except_/lets-talk-about-failures-with-kubernetes-hamburg-meetup', module: 'M8' },
  { company: 'Zalando',       title: 'How to crash your cluster (talk)', url: 'https://www.youtube.com/watch?v=LpFApeaGv7A', module: 'M7' },
  { company: 'Zalando',       title: 'Kubernetes failure stories (talk)', url: 'https://www.youtube.com/watch?v=6sDTB4eV4F8', module: 'M8' },
  { company: 'Airbnb',        title: '10 weird ways to blow up your Kubernetes', url: 'https://www.youtube.com/watch?v=FrQ8Lwm9_j8', module: 'M2' },
  { company: 'Airbnb',        title: '10 MORE weird ways (webhooks, CPU limits, kube2iam)', url: 'https://www.youtube.com/watch?v=4CT0cI62YHk', module: 'M6' },
  { company: 'Airbnb',        title: 'Did Kubernetes make my p95s worse?', url: 'https://www.youtube.com/watch?v=QXApVwRBeys', module: 'M2' },
  { company: 'Skyscanner',    title: 'A couple of characters brought down our site', url: 'https://medium.com/@SkyscannerEng/how-a-couple-of-characters-brought-down-our-site-356ccaf1fbc3', module: 'M8' },
  { company: 'Skyscanner',    title: 'One templating line, clusters in pain', url: 'https://medium.com/@SkyscannerEng/misunderstanding-the-behaviour-of-one-templating-line-and-the-pain-it-caused-our-k8s-clusters-a420f30a99f1', module: 'M4' },
  { company: 'Adevinta',      title: 'Kubernetes made my latency 10× higher', url: 'https://srvaroa.github.io/kubernetes/migration/latency/dns/java/aws/microservices/2019/10/22/kubernetes-added-a-0-to-my-latency.html', module: 'M4' },
  { company: 'Preply',        title: 'DNS postmortem — conntrack races', url: 'https://medium.com/preply-engineering/dns-postmortem-e169efd45afd', module: 'M4' },
  { company: 'loveholidays',  title: 'When GKE ran out of IP addresses', url: 'https://deploy.live/blog/when-gke-ran-out-of-ip-addresses/', module: 'M4' },
  { company: 'loveholidays',  title: 'The shipwreck of a GKE cluster upgrade', url: 'https://deploy.live/blog/the-shipwreck-of-gke-cluster-upgrade/', module: 'M8' },
  { company: 'MindTickle',    title: 'The case of the missing packet (EKS CNI)', url: 'https://yashmehrotra.com/post/2020-03-16-case-of-missing-packet/', module: 'M4' },
  { company: 'MindTickle',    title: 'Intermittent delays — conntrack DNAT races, musl vs libc', url: 'https://medium.com/techmindtickle/intermittent-delays-in-kubernetes-e9de8239e2fa', module: 'M4' },
  { company: 'Civis Analytics', title: 'How we broke (and fixed) our Kubernetes cluster', url: 'https://medium.com/civis-analytics/https-medium-com-civis-analytics-breaking-kubernetes-how-we-broke-and-fixed-our-k8s-cluster-adfa6fbade61', module: 'M8' },
  { company: 'Xing',          title: 'Moving to Kubernetes: the bad and the ugly', url: 'https://www.youtube.com/watch?v=MoIdU0J0f0E', module: 'M8' },
  { company: 'Nordstrom',     title: '101 ways to break and recover a cluster', url: 'https://www.youtube.com/watch?v=xZO9nx6GBu0', module: 'M7' },
  { company: 'FREE NOW',      title: 'New K8s workers unable to join cluster', url: 'https://github.com/freenowtech/postmortems/blob/master/2019-09-19%20-%20New%20K8s%20workers%20unable%20to%20join%20cluster.pdf', module: 'M8' },
  { company: 'Neon',          title: 'IP exhaustion + control-plane overload (repeat incident)', url: 'https://neon.com/blog/postmortem-delayed-start-compute-operations', module: 'M4' },
  { company: 'Chick-fil-A',   title: 'Bare-metal k3s in 2,800+ edge clusters', url: 'https://medium.com/chick-fil-atech/bare-metal-k8s-clustering-at-chick-fil-a-scale-929a0e6d29e5', module: 'M8' },
  { company: 'Google Cloud',  title: '2019 maintenance automation descheduled the network control plane', url: 'https://status.cloud.google.com/incident/cloud-networking/19009', module: 'M7' },
];
