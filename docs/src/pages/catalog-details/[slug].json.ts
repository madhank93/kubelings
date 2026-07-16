// Per-lesson detail, prerendered as static JSON and fetched by the /catalog
// modal on open. Keeping it out of catalog.astro stops 107 problem statements
// and 25 write-ups (~300KB of HTML) from loading with the table.
import type { APIRoute } from 'astro';
import { CATALOG } from '../../data/catalog';
import { FURTHER_READING } from '../../data/further-reading';

type MdModule = {
  frontmatter: Record<string, unknown>;
  // Astro 5 resolves compiled markdown asynchronously.
  compiledContent: () => Promise<string>;
};

// Astro compiles these to HTML at build time; key them by bare filename.
const byName = (glob: Record<string, unknown>) =>
  Object.fromEntries(
    Object.entries(glob).map(([path, mod]) => [
      path.split('/').pop()!.replace(/\.md$/, ''),
      mod as MdModule,
    ])
  );

const DETAILS = byName(import.meta.glob('../../data/lesson-details/*.md', { eager: true }));
const WRITEUPS = byName(import.meta.glob('../../data/incidents/*.md', { eager: true }));

export function getStaticPaths() {
  const slugs = new Set<string>();
  for (const e of CATALOG) {
    if (e.detail || e.writeUp) slugs.add(e.slug);
  }
  // Write-ups for cited incidents that have no lesson of their own — the
  // "further reading" rows open a modal too.
  for (const r of FURTHER_READING) {
    if (r.writeUp) slugs.add(r.writeUp);
  }
  return [...slugs].map((slug) => ({ params: { slug } }));
}

export const GET: APIRoute = async ({ params }) => {
  const slug = params.slug!;
  const entry = CATALOG.find((e) => e.slug === slug);
  const reading = FURTHER_READING.find((r) => r.writeUp === slug);
  const writeUpName = entry?.writeUp ?? reading?.writeUp;
  const writeUp = writeUpName ? WRITEUPS[writeUpName] : undefined;
  const detail = entry?.detail ? DETAILS[slug] : undefined;

  return new Response(
    JSON.stringify({
      slug,
      problem: detail ? await detail.compiledContent() : null,
      writeUp: writeUp ? await writeUp.compiledContent() : null,
      writeUpTitle: (writeUp?.frontmatter.title as string) ?? null,
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );
};
