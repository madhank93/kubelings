// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
  site: 'https://kubelings.madhan.app',
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
            { label: 'The Curriculum', slug: 'guides/curriculum' },
            { label: 'The TUI', slug: 'guides/tui' },
            { label: 'Lessons', slug: 'guides/lessons' },
            { label: 'CLI', slug: 'guides/cli' },
          ],
        },
        {
          label: 'Incidents',
          items: [
            { label: 'Incident Library', slug: 'reference/incident-library' },
            {
              label: 'Case Studies',
              autogenerate: { directory: 'incidents' },
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
