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
        'Learn Kubernetes the rustlings way — fix broken-on-purpose clusters until an automated check passes.',
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
