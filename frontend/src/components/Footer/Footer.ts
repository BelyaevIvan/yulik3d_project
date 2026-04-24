import { renderTemplate } from '@/utils/template';
import { footerTemplate } from './Footer.template';
import { config } from '@/utils/config';
import './Footer.scss';

export class Footer {
  constructor(private root: HTMLElement) {}
  render(): void {
    this.root.innerHTML = renderTemplate(footerTemplate, {
      contact: config.contact,
      year: new Date().getFullYear(),
    });
  }
}
