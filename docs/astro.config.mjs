// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
  site: 'https://kubelings.madhan.app',
  // The Incident Library folded into /catalog (cited sources live on each row +
  // a "further reading" appendix). Keep the old URL alive.
  redirects: {
    '/reference/incident-library/': '/catalog/',
  },
  integrations: [
    starlight({
      title: 'Kubelings',
      logo: { src: './src/assets/kubernetes.svg', replacesTitle: false },
      description:
        'Learn Kubernetes the rustlings way — fix broken-on-purpose clusters until an automated check passes. By Madhan Kumaravelu.',
      head: [
        {
          tag: 'meta',
          attrs: { name: 'author', content: 'Madhan Kumaravelu' },
        },
        {
          tag: 'meta',
          attrs: {
            name: 'keywords',
            content:
              'Kubernetes, learn Kubernetes, Kubernetes hands-on labs, rustlings, kubelings, CKA, CKAD, CKS, Kubernetes troubleshooting, Kubernetes incidents, kind, iximiuz labs, Madhan Kumaravelu, madhank93',
          },
        },
        {
          tag: 'script',
          attrs: { type: 'application/ld+json' },
          content: JSON.stringify({
            '@context': 'https://schema.org',
            '@graph': [
              {
                '@type': 'Person',
                '@id': 'https://kubelings.madhan.app/#author',
                name: 'Madhan Kumaravelu',
                alternateName: 'madhank93',
                url: 'https://madhan.app',
                sameAs: ['https://github.com/madhank93'],
                knowsAbout: [
                  'Kubernetes',
                  'DevOps',
                  'Site Reliability Engineering',
                  'Platform Engineering',
                  'Cloud Native',
                ],
              },
              {
                '@type': 'Course',
                '@id': 'https://kubelings.madhan.app/#course',
                name: 'Kubelings — Learn Kubernetes the Rustlings Way',
                description:
                  '107 hands-on Kubernetes lessons across 10 modules: fix broken-on-purpose clusters until an automated check passes. Includes 40+ real, cited production incidents. Runs on iximiuz Labs and locally on kind.',
                url: 'https://kubelings.madhan.app',
                provider: { '@id': 'https://kubelings.madhan.app/#author' },
                author: { '@id': 'https://kubelings.madhan.app/#author' },
                isAccessibleForFree: true,
                educationalLevel: 'Beginner to Advanced',
                teaches:
                  'Kubernetes troubleshooting, workloads, networking, security, internals, observability, SRE, platform engineering (CKA/CKAD/CKS aligned)',
                hasCourseInstance: [
                  {
                    '@type': 'CourseInstance',
                    courseMode: 'online',
                    location: {
                      '@type': 'VirtualLocation',
                      url: 'https://labs.iximiuz.com/courses/kubelings-dbd840c8',
                    },
                  },
                ],
                offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
              },
              {
                '@type': 'WebSite',
                '@id': 'https://kubelings.madhan.app/#website',
                url: 'https://kubelings.madhan.app',
                name: 'Kubelings',
                author: { '@id': 'https://kubelings.madhan.app/#author' },
                publisher: { '@id': 'https://kubelings.madhan.app/#author' },
              },
            ],
          }),
        },
      ],
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/madhank93/kubelings' },
      ],
      editLink: {
        baseUrl: 'https://github.com/madhank93/kubelings/edit/main/docs/',
      },
      customCss: ['./src/styles/custom.css'],
      sidebar: [
        {
          label: 'Start Here',
          items: [
            { label: 'Introduction', slug: 'introduction' },
            { label: 'Getting Started', slug: 'getting-started' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { label: 'Catalog', link: '/catalog' },
            { label: 'The Curriculum', slug: 'guides/curriculum' },
            { label: 'The TUI', slug: 'guides/tui' },
            { label: 'CLI', slug: 'guides/cli' },
            { label: 'Authoring lessons', slug: 'guides/lessons' },
          ],
        },
        {
          label: 'Case Studies',
          items: [
            {
              label: 'Real incidents',
              collapsed: true,
              items: [
                { label: 'Algolia — Black Friday', slug: 'incidents/algolia-black-friday' },
                { label: 'Blue Matador — node OOM', slug: 'incidents/bluematador-node-oom' },
                { label: 'CircleCI — version skew', slug: 'incidents/circleci-version-skew' },
                { label: 'loveholidays — conntrack', slug: 'incidents/conntrack-exhaustion' },
                { label: 'Datadog — Cilium routes', slug: 'incidents/datadog-cilium-routes' },
                { label: 'Grafana — priority preemption', slug: 'incidents/grafana-priority-preemption' },
                { label: 'Heroku — host update', slug: 'incidents/heroku-host-update' },
                { label: 'Jetstack — webhook outage', slug: 'incidents/jetstack-webhook-outage' },
                { label: 'JW Player — cryptominer', slug: 'incidents/jwplayer-cryptominer' },
                { label: 'Monzo — cascade', slug: 'incidents/monzo-cascade' },
                { label: 'Moonlight — same node', slug: 'incidents/moonlight-same-node' },
                { label: 'OpenAI — telemetry cascade', slug: 'incidents/openai-telemetry-cascade' },
                { label: 'Ravelin — graceful shutdown', slug: 'incidents/ravelin-graceful-shutdown' },
                { label: 'Reddit — Pi-Day', slug: 'incidents/reddit-piday' },
                { label: 'Spotify — cluster delete', slug: 'incidents/spotify-delete' },
                { label: 'Target — cascade', slug: 'incidents/target-cascade' },
                { label: 'Tinder — scale migration', slug: 'incidents/tinder-scale-migration' },
                { label: 'Zalando — DNS outage', slug: 'incidents/zalando-dns-outage' },
              ],
            },
            {
              label: 'Patterns',
              collapsed: true,
              items: [
                { label: 'Ghost endpoints', slug: 'incidents/pattern-ghost-endpoints' },
                { label: 'Namespace stuck Terminating', slug: 'incidents/pattern-namespace-terminating' },
                { label: 'PVC stuck Terminating', slug: 'incidents/pattern-pvc-terminating' },
                { label: 'Readiness probe flap', slug: 'incidents/pattern-readiness-flap' },
                { label: 'Rolling-update deadlock', slug: 'incidents/pattern-rolling-update-deadlock' },
                { label: 'Secret not reloaded', slug: 'incidents/pattern-secret-not-reloaded' },
                { label: 'Zombie CronJobs', slug: 'incidents/pattern-zombie-cronjobs' },
              ],
            },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'Architecture', slug: 'reference/architecture' },
            { label: 'Security', slug: 'reference/security' },
          ],
        },
      ],
    }),
  ],
});
