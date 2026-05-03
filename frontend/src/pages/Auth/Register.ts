import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import { authStore } from '@/store/auth';
import { renderTemplate } from '@/utils/template';
import { toast } from '@/components/Toast/Toast';
import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
import { mountCaptcha, type CaptchaHandle } from '@/components/Captcha/Captcha';
import './Auth.scss';

const RESEND_COOLDOWN_SEC = 60;

const tpl = `
<div class="auth">
  <div class="auth__card">
    <h1 class="auth__title">Регистрация</h1>
    <p class="auth__sub">Создайте аккаунт за минуту</p>
    <form id="regForm">
      <div id="regErr" style="display:none" class="auth__error"></div>
      <div class="auth__field">
        <label class="auth__label" for="email">Email</label>
        <input class="auth__input" type="email" id="email" name="email" required autocomplete="email" />
      </div>
      <div class="auth__field">
        <label class="auth__label" for="full_name">Имя и фамилия</label>
        <input class="auth__input" type="text" id="full_name" name="full_name" required autocomplete="name" />
      </div>
      <div class="auth__field">
        <label class="auth__label" for="phone">Телефон (необязательно)</label>
        <input class="auth__input" type="tel" id="phone" name="phone" autocomplete="tel" placeholder="+7 999 ..." />
      </div>
      <div class="auth__field">
        <label class="auth__label" for="password">Пароль</label>
        <input class="auth__input" type="password" id="password" name="password" required minlength="8" autocomplete="new-password" />
      </div>
      <div class="auth__field">
        <div id="regCaptcha"></div>
      </div>
      <button type="submit" class="auth__submit" id="regSubmit">Создать аккаунт</button>
    </form>
    <div class="auth__alt">
      Уже есть аккаунт? <a href="/login" data-link>Войти</a>
    </div>
  </div>
</div>
`;

const tplPending = `
<div class="auth">
  <div class="auth__card">
    <h1 class="auth__title">Подтвердите email</h1>
    <p class="auth__sub">
      Спасибо за регистрацию! Мы отправили письмо на <strong>{{email}}</strong>.<br>
      Перейдите по ссылке из письма, чтобы подтвердить email и оформлять заказы.
    </p>
    <div class="auth__success" style="display:none" id="pendingMsg"></div>
    <div class="auth__field">
      <div id="pendingCaptcha"></div>
    </div>
    <button type="button" class="auth__submit" id="resendBtn">Отправить ссылку повторно</button>
    <div class="auth__alt" style="margin-top:12px;">
      <a href="/profile" data-link>В личный кабинет</a>
    </div>
    <div class="auth__alt">
      <a href="/" data-link>← На главную</a>
    </div>
  </div>
</div>
`;

export class RegisterPage {
  constructor(private root: HTMLElement, private query: URLSearchParams) {}

  private captcha: CaptchaHandle | null = null;

  render(): void {
    setPageMeta({ title: 'Регистрация', noindex: true });
    clearProductJsonLd();
    this.root.innerHTML = renderTemplate(tpl, {});
    const form = this.root.querySelector<HTMLFormElement>('#regForm');
    const err = this.root.querySelector<HTMLElement>('#regErr');
    const btn = this.root.querySelector<HTMLButtonElement>('#regSubmit');
    const captchaHost = this.root.querySelector<HTMLElement>('#regCaptcha');
    if (captchaHost) {
      mountCaptcha(captchaHost).then((h) => { this.captcha = h; });
    }

    form?.addEventListener('submit', async (e) => {
      e.preventDefault();
      if (err) err.style.display = 'none';

      // Капча обязательна. Если виджет не успел смонтироваться или юзер
      // не поставил галочку — не отправляем запрос вообще.
      const captchaToken = this.captcha?.getToken() || '';
      if (!captchaToken) {
        toast.error('Подтвердите, что вы не робот');
        return;
      }

      const fd = new FormData(form);
      const phone = String(fd.get('phone') || '').trim();
      btn!.disabled = true;
      try {
        const u = await authApi.register({
          email: String(fd.get('email')),
          password: String(fd.get('password')),
          full_name: String(fd.get('full_name')),
          phone: phone || undefined,
          captcha_token: captchaToken,
        });
        // Юзер зарегистрирован и залогинен. Сессионный cookie уже стоит,
        // authStore хранит профиль с email_verified=false.
        // Сразу показываем экран «проверьте почту» — query-параметр next
        // (если был) отложим до момента подтверждения email.
        authStore.setUser(u);
        toast.success('Аккаунт создан!');
        this.renderPending(u.email);
      } catch (e) {
        if (err) {
          err.textContent = e instanceof ApiError ? e.message : 'Ошибка регистрации';
          err.style.display = 'block';
        }
        // Сбрасываем капчу — токен одноразовый, для повторного submit нужен новый.
        this.captcha?.reset();
      } finally {
        btn!.disabled = false;
      }
    });
  }

  private renderPending(email: string): void {
    this.root.innerHTML = renderTemplate(tplPending, { email });
    const btn = this.root.querySelector<HTMLButtonElement>('#resendBtn')!;
    const msg = this.root.querySelector<HTMLElement>('#pendingMsg');
    const captchaHost = this.root.querySelector<HTMLElement>('#pendingCaptcha');

    // Заново монтируем виджет — после регистрации старый widgetId уже использован.
    let pendingCaptcha: CaptchaHandle | null = null;
    if (captchaHost) {
      mountCaptcha(captchaHost).then((h) => { pendingCaptcha = h; });
    }

    btn.addEventListener('click', async () => {
      const captchaToken = pendingCaptcha?.getToken() || '';
      if (!captchaToken) {
        toast.error('Подтвердите, что вы не робот');
        return;
      }
      btn.disabled = true;
      try {
        await authApi.emailVerifyResend(email, captchaToken);
        if (msg) {
          msg.textContent = 'Письмо отправлено повторно. Проверьте почту (включая папку «Спам»).';
          msg.style.display = 'block';
        }
        pendingCaptcha?.reset();
        startCooldown(btn, RESEND_COOLDOWN_SEC);
      } catch (e) {
        btn.disabled = false;
        pendingCaptcha?.reset();
        toast.error(e instanceof ApiError ? e.message : 'Не удалось отправить письмо');
      }
    });
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
