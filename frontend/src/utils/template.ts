import Handlebars from 'handlebars';

const cache = new Map<string, HandlebarsTemplateDelegate>();

export function compileTemplate(src: string): HandlebarsTemplateDelegate {
  const cached = cache.get(src);
  if (cached) return cached;
  const compiled = Handlebars.compile(src);
  cache.set(src, compiled);
  return compiled;
}

export function renderTemplate(src: string, ctx: Record<string, any> = {}): string {
  return compileTemplate(src)(ctx);
}

// Helpers
Handlebars.registerHelper('formatPrice', (price: number) => {
  if (price == null) return '';
  return new Intl.NumberFormat('ru-RU').format(price) + ' ₽';
});

Handlebars.registerHelper('formatDate', (iso: string) => {
  if (!iso) return '';
  try {
    const d = new Date(iso);
    return d.toLocaleDateString('ru-RU', {
      day: '2-digit', month: '2-digit', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  } catch { return iso; }
});

Handlebars.registerHelper('eq', (a: unknown, b: unknown) => a === b);
Handlebars.registerHelper('gt', (a: number, b: number) => a > b);
Handlebars.registerHelper('and', (a: any, b: any) => Boolean(a && b));
Handlebars.registerHelper('or', (a: any, b: any) => Boolean(a || b));
Handlebars.registerHelper('not', (a: any) => !a);

Handlebars.registerHelper('pluralize', (n: number, one: string, few: string, many: string) => {
  const m10 = n % 10, m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return few;
  return many;
});

// Статусы заказа на русском
const statusLabels: Record<string, string> = {
  created: 'Создан',
  confirmed: 'Подтверждён',
  manufacturing: 'На изготовлении',
  delivering: 'В доставке',
  completed: 'Завершён',
  cancelled: 'Отменён',
};
Handlebars.registerHelper('orderStatusLabel', (s: string) => statusLabels[s] || s);

Handlebars.registerHelper('orderStatusClass', (s: string) => {
  const map: Record<string, string> = {
    created: 'status--created',
    confirmed: 'status--confirmed',
    manufacturing: 'status--manufacturing',
    delivering: 'status--delivering',
    completed: 'status--completed',
    cancelled: 'status--cancelled',
  };
  return map[s] || '';
});
