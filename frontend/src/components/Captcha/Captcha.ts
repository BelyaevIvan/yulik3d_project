// Yandex SmartCaptcha — visible-виджет (чекбокс «Я не робот»).
//
// Использование:
//
//   const captcha = await mountCaptcha(host);
//   const token = captcha.getToken();
//   if (!token) { toast.error('Поставьте галочку «Я не робот»'); return; }
//   // дальше — captcha_token в payload запроса
//   captcha.reset();   // вызвать после успешного submit'а или после ошибки,
//                      // чтобы виджет был готов к следующему использованию
//
// Скрипт https://smartcaptcha.cloud.yandex.ru/captcha.js подключён глобально
// в index.html с атрибутом defer. window.smartCaptcha появляется после загрузки.

declare global {
  interface Window {
    smartCaptcha?: {
      render(
        container: HTMLElement | string,
        params: {
          sitekey: string;
          invisible?: boolean;
          callback?: (token: string) => void;
          hl?: 'ru' | 'en';
        }
      ): number;
      execute(widgetId?: number): void;
      reset(widgetId?: number): void;
      destroy(widgetId?: number): void;
      getResponse(widgetId?: number): string;
    };
  }
}

const CAPTCHA_KEY = import.meta.env.VITE_YANDEX_CAPTCHA_KEY as string | undefined;

export interface CaptchaHandle {
  getToken(): string;
  reset(): void;
  destroy(): void;
}

// waitForSmartCaptcha — ждёт, пока загрузится скрипт Yandex.
// Если за timeoutMs не дождались — ошибка (видимо, скрипт заблокирован).
async function waitForSmartCaptcha(timeoutMs = 5000): Promise<NonNullable<Window['smartCaptcha']>> {
  const start = Date.now();
  while (!window.smartCaptcha) {
    if (Date.now() - start > timeoutMs) {
      throw new Error('SmartCaptcha script not loaded (blocked by adblock?)');
    }
    await new Promise((r) => setTimeout(r, 50));
  }
  return window.smartCaptcha;
}

// mountCaptcha — рендерит visible-виджет в указанный контейнер.
// Если CAPTCHA_KEY не задан — пишет плейсхолдер и отдаёт null.
export async function mountCaptcha(host: HTMLElement): Promise<CaptchaHandle | null> {
  if (!CAPTCHA_KEY) {
    host.innerHTML =
      '<div style="color:#a0a0c0;font-size:13px;padding:8px;border:1px dashed #555;border-radius:6px;">' +
      'Защита от ботов отключена (VITE_YANDEX_CAPTCHA_KEY не задан).' +
      '</div>';
    return null;
  }

  let smartCaptcha: NonNullable<Window['smartCaptcha']>;
  try {
    smartCaptcha = await waitForSmartCaptcha();
  } catch (e) {
    host.innerHTML =
      '<div style="color:#f44;font-size:13px;padding:8px;border:1px dashed #f44;border-radius:6px;">' +
      'Не удалось загрузить капчу. Проверьте, что блокировщик рекламы не блокирует Yandex, и обновите страницу.' +
      '</div>';
    return null;
  }

  const widgetId = smartCaptcha.render(host, {
    sitekey: CAPTCHA_KEY,
    hl: 'ru',
  });

  return {
    getToken: () => smartCaptcha.getResponse(widgetId) || '',
    reset: () => smartCaptcha.reset(widgetId),
    destroy: () => smartCaptcha.destroy(widgetId),
  };
}
