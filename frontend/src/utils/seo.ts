// SEO-метаданные. Вызывается из каждой страницы при рендере.
// Обновляет <title>, <meta name="description">, Open Graph (og:*) и Twitter Card теги.

const SITE_NAME = 'Магазин 3D-печати YULIK3D';
const SITE_URL = 'https://yulik3d.ru';
const DEFAULT_DESCRIPTION =
  'Фигурки и макеты на заказ. 3D-печать персонажей из игр, фильмов, аниме. ' +
  'Декор, аксессуары, кастомизация. Доставка по России.';
const DEFAULT_IMAGE = SITE_URL + '/logo.svg';

export interface PageMeta {
  title?: string;          // короткое название страницы (без " — YULIK3D" — добавится сам)
  description?: string;    // 80–160 символов
  image?: string;          // абсолютный URL картинки (для шаринга)
  type?: 'website' | 'product' | 'article';
  noindex?: boolean;       // true → запретить индексацию (для админки, логина и т.п.)
}

function setMetaTag(attr: 'name' | 'property', key: string, value: string): void {
  let el = document.head.querySelector<HTMLMetaElement>(`meta[${attr}="${key}"]`);
  if (!el) {
    el = document.createElement('meta');
    el.setAttribute(attr, key);
    document.head.appendChild(el);
  }
  el.setAttribute('content', value);
}

function setLink(rel: string, href: string): void {
  let el = document.head.querySelector<HTMLLinkElement>(`link[rel="${rel}"]`);
  if (!el) {
    el = document.createElement('link');
    el.setAttribute('rel', rel);
    document.head.appendChild(el);
  }
  el.setAttribute('href', href);
}

export function setPageMeta(meta: PageMeta = {}): void {
  const title = meta.title ? `${meta.title} — ${SITE_NAME}` : SITE_NAME;
  const description = (meta.description || DEFAULT_DESCRIPTION).trim();
  const image = meta.image || DEFAULT_IMAGE;
  const type = meta.type || 'website';
  const url = SITE_URL + window.location.pathname + window.location.search;

  document.title = title;
  setMetaTag('name', 'description', description);

  // Канонический URL
  setLink('canonical', url);

  // Open Graph (для VK, Telegram, Facebook)
  setMetaTag('property', 'og:title', title);
  setMetaTag('property', 'og:description', description);
  setMetaTag('property', 'og:image', image);
  setMetaTag('property', 'og:url', url);
  setMetaTag('property', 'og:type', type);
  setMetaTag('property', 'og:site_name', SITE_NAME);
  setMetaTag('property', 'og:locale', 'ru_RU');

  // Twitter Card
  setMetaTag('name', 'twitter:card', 'summary_large_image');
  setMetaTag('name', 'twitter:title', title);
  setMetaTag('name', 'twitter:description', description);
  setMetaTag('name', 'twitter:image', image);

  // Robots
  setMetaTag('name', 'robots', meta.noindex ? 'noindex, nofollow' : 'index, follow');
}

/**
 * JSON-LD структурированные данные. Используется на странице товара,
 * чтобы Google показывал в выдаче цену и наличие.
 */
export function setProductJsonLd(product: {
  name: string;
  description: string;
  image: string[];
  price: number;
  availability: 'in stock' | 'out of stock';
  url: string;
  sku: string;
}): void {
  // Удалить предыдущий JSON-LD, если был
  const old = document.head.querySelector('script[data-jsonld="product"]');
  if (old) old.remove();

  const data = {
    '@context': 'https://schema.org/',
    '@type': 'Product',
    name: product.name,
    description: product.description,
    image: product.image,
    sku: product.sku,
    brand: { '@type': 'Brand', name: SITE_NAME },
    offers: {
      '@type': 'Offer',
      url: product.url,
      priceCurrency: 'RUB',
      price: product.price,
      availability:
        product.availability === 'in stock'
          ? 'https://schema.org/InStock'
          : 'https://schema.org/OutOfStock',
    },
  };

  const script = document.createElement('script');
  script.type = 'application/ld+json';
  script.dataset.jsonld = 'product';
  script.textContent = JSON.stringify(data);
  document.head.appendChild(script);
}

export function clearProductJsonLd(): void {
  document.head.querySelector('script[data-jsonld="product"]')?.remove();
}
