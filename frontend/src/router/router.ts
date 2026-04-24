export interface Route {
  path: string;
  handler: (params: Record<string, string>, query: URLSearchParams) => void | Promise<void>;
}

class Router {
  private routes: Route[] = [];
  private notFoundHandler: (() => void) | null = null;

  addRoute(path: string, handler: Route['handler']): void {
    this.routes.push({ path, handler });
  }

  setNotFound(fn: () => void): void {
    this.notFoundHandler = fn;
  }

  navigate(path: string): void {
    window.history.pushState({}, '', path);
    this.resolve();
  }

  replace(path: string): void {
    window.history.replaceState({}, '', path);
    this.resolve();
  }

  private match(pathname: string): { route: Route; params: Record<string, string> } | null {
    for (const r of this.routes) {
      const names: string[] = [];
      const re = new RegExp('^' + r.path.replace(/:(\w+)/g, (_, n) => {
        names.push(n);
        return '([^/]+)';
      }) + '$');
      const m = pathname.match(re);
      if (m) {
        const params: Record<string, string> = {};
        names.forEach((n, i) => (params[n] = decodeURIComponent(m[i + 1])));
        return { route: r, params };
      }
    }
    return null;
  }

  resolve(): void {
    const url = new URL(window.location.href);
    const found = this.match(url.pathname);
    if (found) {
      try {
        const r = found.route.handler(found.params, url.searchParams);
        if (r instanceof Promise) r.catch((e) => console.error('route handler:', e));
      } catch (e) {
        console.error('route handler:', e);
      }
      window.scrollTo({ top: 0 });
      return;
    }
    if (this.notFoundHandler) {
      this.notFoundHandler();
      window.scrollTo({ top: 0 });
    }
  }

  start(): void {
    window.addEventListener('popstate', () => this.resolve());
    document.addEventListener('click', (e) => {
      const a = (e.target as HTMLElement).closest('a[data-link]') as HTMLAnchorElement | null;
      if (!a) return;
      const href = a.getAttribute('href');
      if (!href || href.startsWith('http') || href.startsWith('mailto:') || href.startsWith('tel:')) return;
      e.preventDefault();
      this.navigate(href);
    });
    this.resolve();
  }

  currentQuery(): URLSearchParams {
    return new URL(window.location.href).searchParams;
  }
  currentPath(): string {
    return window.location.pathname;
  }
}

export const router = new Router();
