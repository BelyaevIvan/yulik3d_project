// Универсальная модалка с body-content и optional footer.
import { renderTemplate } from '@/utils/template';
import './Modal.scss';

const tpl = `
<div class="modal-overlay" data-overlay>
  <div class="modal" role="dialog" aria-modal="true">
    <div class="modal__header">
      <h3 class="modal__title">{{title}}</h3>
      <button class="modal__close" data-close aria-label="Закрыть">×</button>
    </div>
    <div class="modal__body">{{{body}}}</div>
    {{#if footer}}<div class="modal__footer">{{{footer}}}</div>{{/if}}
  </div>
</div>
`;

export interface ModalOptions {
  title: string;
  body: string;        // HTML
  footer?: string;     // HTML, обычно с кнопками
  onMount?: (root: HTMLElement) => void;
  onClose?: () => void;
}

class ModalManager {
  private current: HTMLElement | null = null;
  private opts: ModalOptions | null = null;

  open(opts: ModalOptions): void {
    this.close();
    const wrap = document.createElement('div');
    wrap.innerHTML = renderTemplate(tpl, opts);
    document.body.appendChild(wrap);
    this.current = wrap;
    this.opts = opts;
    document.body.style.overflow = 'hidden';

    wrap.querySelector('[data-close]')?.addEventListener('click', () => this.close());
    wrap.querySelector('[data-overlay]')?.addEventListener('click', (e) => {
      if ((e.target as HTMLElement).hasAttribute('data-overlay')) this.close();
    });

    if (opts.onMount) opts.onMount(wrap);
  }

  close(): void {
    if (!this.current) return;
    this.current.remove();
    this.current = null;
    document.body.style.overflow = '';
    if (this.opts?.onClose) this.opts.onClose();
    this.opts = null;
  }
}

export const modal = new ModalManager();
