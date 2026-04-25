// Простые toast-уведомления.
import './Toast.scss';

type Kind = 'info' | 'success' | 'error';

class Toast {
  private container: HTMLElement | null = null;

  private ensureContainer(): HTMLElement {
    if (this.container) return this.container;
    const el = document.createElement('div');
    el.className = 'toast-container';
    document.body.appendChild(el);
    this.container = el;
    return el;
  }

  show(message: string, kind: Kind = 'info', timeoutMs = 3500): void {
    const c = this.ensureContainer();
    const t = document.createElement('div');
    t.className = `toast toast--${kind}`;
    t.textContent = message;
    c.appendChild(t);
    requestAnimationFrame(() => t.classList.add('toast--visible'));
    setTimeout(() => {
      t.classList.remove('toast--visible');
      setTimeout(() => t.remove(), 300);
    }, timeoutMs);
  }

  success(m: string): void { this.show(m, 'success'); }
  error(m: string): void { this.show(m, 'error', 5000); }
  info(m: string): void { this.show(m, 'info'); }
}

export const toast = new Toast();
