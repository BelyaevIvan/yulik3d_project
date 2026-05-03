import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import { authStore } from '@/store/auth';
import { toast } from '@/components/Toast/Toast';
import { modal } from '@/components/Modal/Modal';
import { mountCaptcha } from '@/components/Captcha/Captcha';
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

// onResendClick — открывает модалку с капчей. После прохождения капчи
// фронт шлёт запрос на бэк, и при успехе включает cooldown на основной кнопке.
function onResendClick(btn: HTMLButtonElement, email: string): void {
  modal.open({
    title: 'Отправить письмо повторно',
    body: `
      <p style="margin:0 0 12px;color:#a0a0c0;">
        На <strong>${escapeHtml(email)}</strong> будет отправлена новая ссылка для подтверждения email.
      </p>
      <div id="bannerResendCaptcha" style="margin-bottom:12px;"></div>
      <div id="bannerResendErr" style="display:none;color:#f44;font-size:13px;margin-bottom:12px;"></div>
    `,
    footer: `
      <button data-cancel style="padding:8px 16px;background:#141450;color:#fff;border:1px solid #2a2a5a;border-radius:6px;">Отмена</button>
      <button data-send class="admin__btn" style="padding:8px 16px;">Отправить</button>
    `,
    onMount: (root) => {
      const captchaHost = root.querySelector<HTMLElement>('#bannerResendCaptcha')!;
      const errEl = root.querySelector<HTMLElement>('#bannerResendErr');
      let handle: import('@/components/Captcha/Captcha').CaptchaHandle | null = null;
      mountCaptcha(captchaHost).then((h) => { handle = h; });

      root.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
      root.querySelector('[data-send]')?.addEventListener('click', async () => {
        const token = handle?.getToken() || '';
        if (!token) {
          if (errEl) { errEl.textContent = 'Подтвердите, что вы не робот'; errEl.style.display = 'block'; }
          return;
        }
        try {
          await authApi.emailVerifyResend(email, token);
          modal.close();
          toast.success('Письмо отправлено. Проверьте почту (включая папку «Спам»).');
          startCooldown(btn, RESEND_COOLDOWN_SEC);
        } catch (e) {
          handle?.reset();
          const msg = e instanceof ApiError ? e.message : 'Не удалось отправить письмо';
          if (errEl) { errEl.textContent = msg; errEl.style.display = 'block'; }
        }
      });
    },
  });
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
