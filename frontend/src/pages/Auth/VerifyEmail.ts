import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import { authStore } from '@/store/auth';
import { router } from '@/router/router';
import { renderTemplate } from '@/utils/template';
import { toast } from '@/components/Toast/Toast';
import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
import './Auth.scss';

const tplLoading = `
<div class="auth">
  <div class="auth__card">
    <h1 class="auth__title">Подтверждение email</h1>
    <p class="auth__sub">Проверяем ссылку...</p>
  </div>
</div>
`;

const tplError = `
<div class="auth">
  <div class="auth__card">
    <h1 class="auth__title">Ссылка не работает</h1>
    <p class="auth__sub">{{message}}</p>
    <p class="auth__sub">Запросите новую ссылку из личного кабинета — старая ссылка действительна 24 часа и только один раз.</p>
    <div class="auth__alt">
      <a href="/profile" data-link>Перейти в личный кабинет</a>
    </div>
    <div class="auth__alt">
      <a href="/" data-link>← На главную</a>
    </div>
  </div>
</div>
`;

const tplBadToken = `
<div class="auth">
  <div class="auth__card">
    <h1 class="auth__title">Ссылка не работает</h1>
    <p class="auth__sub">В ссылке нет токена подтверждения. Возможно, она была обрезана при копировании.</p>
    <div class="auth__alt">
      <a href="/profile" data-link>Перейти в личный кабинет</a>
    </div>
  </div>
</div>
`;

export class VerifyEmailPage {
  constructor(private root: HTMLElement, private query: URLSearchParams) {}

  async render(): Promise<void> {
    setPageMeta({ title: 'Подтверждение email', noindex: true });
    clearProductJsonLd();

    const token = this.query.get('token') || '';
    if (!token) {
      this.root.innerHTML = renderTemplate(tplBadToken, {});
      return;
    }

    this.root.innerHTML = renderTemplate(tplLoading, {});

    try {
      await authApi.emailVerifyConfirm(token);
      // Успех — обновляем authStore (если юзер залогинен, у него теперь email_verified=true).
      // Если юзер не залогинен — refresh() ничего не подтянет, и это ОК: он просто перейдёт на /login и при следующем заходе увидит свежее состояние.
      await authStore.refresh();
      toast.success('Email подтверждён, спасибо!');
      router.navigate('/profile');
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : 'Ошибка подтверждения';
      this.root.innerHTML = renderTemplate(tplError, { message: msg });
    }
  }
}
