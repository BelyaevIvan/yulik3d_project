import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import { authStore } from '@/store/auth';
import { router } from '@/router/router';
import { renderTemplate } from '@/utils/template';
import { toast } from '@/components/Toast/Toast';
import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
import './Profile.scss';

const tpl = `
<div class="profile">
  <h1 class="profile__title">Профиль</h1>
  <div class="profile__layout">
    <div class="profile__card">
      <h2 class="profile__heading">Личные данные</h2>
      <form id="profileForm">
        <div id="profileErr" style="display:none" class="auth__error"></div>
        <div class="profile__field">
          <label class="profile__label">Email</label>
          <input class="profile__input" type="email" value="{{email}}" disabled />
          <small class="profile__hint">Email не редактируется</small>
        </div>
        <div class="profile__field">
          <label class="profile__label" for="full_name">Имя и фамилия</label>
          <input class="profile__input" type="text" id="full_name" name="full_name" value="{{full_name}}" required />
        </div>
        <div class="profile__field">
          <label class="profile__label" for="phone">Телефон</label>
          <input class="profile__input" type="tel" id="phone" name="phone" value="{{phone}}" />
        </div>
        <button type="submit" class="profile__submit">Сохранить</button>
      </form>
    </div>

    <div class="profile__card">
      <h2 class="profile__heading">Сменить пароль</h2>
      <form id="passwordForm">
        <div id="passErr" style="display:none" class="auth__error"></div>
        <div class="profile__field">
          <label class="profile__label" for="old_password">Старый пароль</label>
          <input class="profile__input" type="password" id="old_password" name="old_password" autocomplete="current-password" required />
        </div>
        <div class="profile__field">
          <label class="profile__label" for="new_password">Новый пароль</label>
          <input class="profile__input" type="password" id="new_password" name="new_password" minlength="8" autocomplete="new-password" required />
        </div>
        <div class="profile__field">
          <label class="profile__label" for="new_password_2">Повторите новый пароль</label>
          <input class="profile__input" type="password" id="new_password_2" name="new_password_2" minlength="8" required />
        </div>
        <button type="submit" class="profile__submit profile__submit--secondary">Сменить пароль</button>
      </form>
    </div>
  </div>
</div>
`;

export class ProfilePage {
  constructor(private root: HTMLElement) {}

  render(): void {
    if (!authStore.isAuthed()) {
      router.navigate('/login?next=/profile');
      return;
    }
    setPageMeta({ title: 'Профиль', noindex: true });
    clearProductJsonLd();
    const u = authStore.getUser()!;
    this.root.innerHTML = renderTemplate(tpl, {
      email: u.email,
      full_name: u.full_name,
      phone: u.phone || '',
    });

    // Profile form
    const pf = this.root.querySelector<HTMLFormElement>('#profileForm');
    const perr = this.root.querySelector<HTMLElement>('#profileErr');
    pf?.addEventListener('submit', async (e) => {
      e.preventDefault();
      if (perr) perr.style.display = 'none';
      const fd = new FormData(pf);
      const phone = String(fd.get('phone') || '').trim();
      try {
        const updated = await authApi.updateMe({
          full_name: String(fd.get('full_name')),
          phone: phone || undefined,
        });
        authStore.setUser(updated);
        toast.success('Профиль обновлён');
      } catch (err) {
        if (perr) {
          perr.textContent = err instanceof ApiError ? err.message : 'Ошибка обновления';
          perr.style.display = 'block';
        }
      }
    });

    // Password form
    const wf = this.root.querySelector<HTMLFormElement>('#passwordForm');
    const werr = this.root.querySelector<HTMLElement>('#passErr');
    wf?.addEventListener('submit', async (e) => {
      e.preventDefault();
      if (werr) werr.style.display = 'none';
      const fd = new FormData(wf);
      const np = String(fd.get('new_password'));
      const np2 = String(fd.get('new_password_2'));
      if (np !== np2) {
        if (werr) { werr.textContent = 'Новые пароли не совпадают'; werr.style.display = 'block'; }
        return;
      }
      try {
        await authApi.updateMe({
          old_password: String(fd.get('old_password')),
          new_password: np,
        });
        toast.success('Пароль изменён');
        wf.reset();
      } catch (err) {
        if (werr) {
          werr.textContent = err instanceof ApiError ? err.message : 'Ошибка смены пароля';
          werr.style.display = 'block';
        }
      }
    });
  }
}
