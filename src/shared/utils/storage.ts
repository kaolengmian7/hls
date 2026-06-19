const PREFIX = 'hls:v1:';

function key(name: string): string {
  return PREFIX + name;
}

export const storage = {
  get<T>(name: string, fallback: T): T {
    try {
      const raw = localStorage.getItem(key(name));
      if (raw == null) return fallback;
      return JSON.parse(raw) as T;
    } catch {
      return fallback;
    }
  },

  set<T>(name: string, value: T): void {
    try {
      localStorage.setItem(key(name), JSON.stringify(value));
    } catch {
      // ignore quota errors
    }
  },

  remove(name: string): void {
    try {
      localStorage.removeItem(key(name));
    } catch {
      // ignore
    }
  },
};
