import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import { renderTemplate } from '@/utils/template';
import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
import { toast } from '@/components/Toast/Toast';
import { mountCaptcha, type CaptchaHandle } from '@/components/Captcha/Captcha';
import './Auth.scss';

const tpl = `
<div class="auth">
  <div class="auth__card">
    <h1 class="auth__title">Восстановление пароля</h1>
    <p class="auth__sub">Введите email, на который зарегистрирован аккаунт. Мы отправим ссылку для смены пароля.</p>
    <form id="forgotForm">
      <div id="forgotMsg" style="display:none" class="auth__success"></div>
      <div id="forgotErr" style="display:none" class="auth__error"></div>
      <div class="auth__field">
        <label class="auth__label" for="email">Email</label>
        <input class="auth__input" type="email" id="email" name="email" required autocomplete="email" />
      </div>
      <div class="auth__field">
        <div id="forgotCaptcha"></div>
      </div>
      <button type="submit" class="auth__submit" id="forgotSubmit">Отправить ссылку</button>
    </form>
    <div class="auth__alt" style="margin-top:12px;">
      <a href="/login" data-link>← Назад к входу</a>
    </div>
  </div>
</div>
`;

export class ForgotPasswordPage {
  private captcha: CaptchaHandle | null = null;

  constructor(private root: HTMLElement) {}

  render(): void {
    setPageMeta({ title: 'Восстановление пароля', noindex: true });
    clearProductJsonLd();
    this.root.innerHTML = renderTemplate(tpl, {});
    const form = this.root.querySelector<HTMLFormElement>('#forgotForm');
    const msg = this.root.querySelector<HTMLElement>('#forgotMsg');
    const err = this.root.querySelector<HTMLElement>('#forgotErr');
    const btn = this.root.querySelector<HTMLButtonElement>('#forgotSubmit');
    const captchaHost = this.root.querySelector<HTMLElement>('#forgotCaptcha');
    if (captchaHost) {
      mountCaptcha(captchaHost).then((h) => { this.captcha = h; });
    }

    form?.addEventListener('submit', async (e) => {
      e.preventDefault();
      if (err) err.style.display = 'none';
      if (msg) msg.style.display = 'none';

      const captchaToken = this.captcha?.getToken() || '';
      if (!captchaToken) {
        toast.error('Подтвердите, что вы не робот');
        return;
      }

      const fd = new FormData(form);
      btn!.disabled = true;
      try {
        await authApi.passwordResetRequest(String(fd.get('email')), captchaToken);
        // Бэкенд ВСЕГДА возвращает 200 (если капча прошла) — не палит существование email.
        if (msg) {
          msg.textContent = 'Если этот email привязан к аккаунту, мы отправили на него ссылку. Проверьте почту (включая папку «Спам») в течение нескольких минут.';
          msg.style.display = 'block';
        }
        form.reset();
        this.captcha?.reset();
      } catch (e) {
        if (err) {
          err.textContent = e instanceof ApiError ? e.message : 'Ошибка отправки';
          err.style.display = 'block';
        }
        this.captcha?.reset();
      } finally {
        btn!.disabled = false;
      }
    });
  }
}
