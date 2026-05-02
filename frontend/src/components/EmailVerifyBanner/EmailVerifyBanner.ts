import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import { authStore } from '@/store/auth';
import { toast } from '@/components/Toast/Toast';
import './EmailVerifyBanner.scss';

const RESEND_COOLDOWN_SEC = 60;

// renderEmailVerifyBanner — вставляет баннер «email не подтверждён» в начало
// переданного контейнера, если пользователь авторизован и email_verified=false.
//
// Если условия не выполнены — ничего не делает (молча).
//
// Использование: вызывается из render() страниц Profile/Cart и т.п. ПОСЛЕ
// того, как они отрендерили свой HTML.
export function renderEmailVerifyBanner(host: HTMLElement): void {
  const user = authStore.getUser();
  if (!user || user.email_verified) return;

  const banner = document.createElement('div');
  banner.className = 'email-verify-banner';
  banner.innerHTML = `
    <div class="email-verify-banner__icon">⚠️</div>
    <div class="email-verify-banner__body">
      <div class="email-verify-banner__title">Email не подтверждён</div>
      <div class="email-verify-banner__text">
        Без подтверждения email вы не сможете оформить заказ. Мы отправили ссылку на <strong>${escapeHtml(user.email)}</strong>.
      </div>
    </div>
    <button class="email-verify-banner__btn" data-act="resend">Отправить ссылку повторно</button>
  `;
  host.prepend(banner);

  const btn = banner.querySelector<HTMLButtonElement>('[data-act="resend"]')!;
  btn.addEventListener('click', () => onResendClick(btn, user.email));
}

async function onResendClick(btn: HTMLButtonElement, email: string): Promise<void> {
  btn.disabled = true;
  try {
    await authApi.emailVerifyResend(email);
    toast.success('Письмо отправлено. Проверьте почту (включая папку «Спам»).');
    startCooldown(btn, RESEND_COOLDOWN_SEC);
  } catch (e) {
    btn.disabled = false;
    if (e instanceof ApiError) toast.error(e.message);
    else toast.error('Не удалось отправить письмо');
  }
}

function startCooldown(btn: HTMLButtonElement, seconds: number): void {
  const original = btn.textContent || 'Отправить ссылку повторно';
  let left = seconds;
  btn.disabled = true;
  btn.textContent = `Отправить ещё раз через ${left} сек`;
  const interval = window.setInterval(() => {
    left -= 1;
    if (left <= 0) {
      window.clearInterval(interval);
      btn.disabled = false;
      btn.textContent = original;
      return;
    }
    btn.textContent = `Отправить ещё раз через ${left} сек`;
  }, 1000);
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) =>
    ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]!)
  );
}
