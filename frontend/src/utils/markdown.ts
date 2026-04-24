import { marked } from 'marked';

marked.setOptions({ breaks: true, gfm: true });

export function renderMarkdown(src: string): string {
  if (!src) return '';
  return marked.parse(src) as string;
}
