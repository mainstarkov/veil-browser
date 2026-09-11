"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

type Tab = {
  id: string;
  title: string;
  url: string;
  history: string[];
  historyIndex: number;
  private: boolean;
};

type PrivacySettings = {
  trackers: boolean;
  cookies: boolean;
  fingerprint: boolean;
  webrtc: boolean;
};

type SessionState = {
  tabs: Tab[];
  activeTabId: string;
  settings: PrivacySettings;
  blocked: number;
};

const SESSION_KEY = "veil-browser-session-v1";

const defaultSettings: PrivacySettings = {
  trackers: true,
  cookies: true,
  fingerprint: true,
  webrtc: true,
};

const createTab = (isPrivate = false): Tab => ({
  id: crypto.randomUUID(),
  title: isPrivate ? "Приватная вкладка" : "Новая вкладка",
  url: "veil://start",
  history: ["veil://start"],
  historyIndex: 0,
  private: isPrivate,
});

const safeSession = (): SessionState => {
  const firstTab = createTab();
  return {
    tabs: [firstTab],
    activeTabId: firstTab.id,
    settings: defaultSettings,
    blocked: 47,
  };
};

function normalizeAddress(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return "veil://start";
  if (/^[a-z]+:\/\//i.test(trimmed)) return trimmed;
  if (/^[\w-]+(\.[\w-]+)+/i.test(trimmed)) return `https://${trimmed}`;
  return `https://duckduckgo.com/?q=${encodeURIComponent(trimmed)}`;
}

function displayHost(url: string) {
  if (url === "veil://start") return "Новая вкладка";
  try {
    return new URL(url).hostname.replace(/^www\./, "") || url;
  } catch {
    return url;
  }
}

export default function Home() {
  const [session, setSession] = useState<SessionState | null>(null);
  const [address, setAddress] = useState("");
  const [panel, setPanel] = useState<"privacy" | "history" | null>(null);

  useEffect(() => {
    let restored: SessionState;
    try {
      const stored = sessionStorage.getItem(SESSION_KEY);
      restored = stored ? JSON.parse(stored) : safeSession();
    } catch {
      restored = safeSession();
    }
    const frame = requestAnimationFrame(() => setSession(restored));
    return () => cancelAnimationFrame(frame);
  }, []);

  useEffect(() => {
    if (session) sessionStorage.setItem(SESSION_KEY, JSON.stringify(session));
  }, [session]);

  const activeTab = useMemo(
    () => session?.tabs.find((tab) => tab.id === session.activeTabId),
    [session],
  );

  useEffect(() => {
    if (!activeTab) return;
    const frame = requestAnimationFrame(() =>
      setAddress(activeTab.url === "veil://start" ? "" : activeTab.url),
    );
    return () => cancelAnimationFrame(frame);
  }, [activeTab]);

  if (!session || !activeTab) {
    return <main className="loading-screen">Запускаем защищённую сессию…</main>;
  }

  const updateActive = (updater: (tab: Tab) => Tab) => {
    setSession((current) =>
      current
        ? {
            ...current,
            tabs: current.tabs.map((tab) =>
              tab.id === current.activeTabId ? updater(tab) : tab,
            ),
          }
        : current,
    );
  };

  const navigate = (raw: string) => {
    const url = normalizeAddress(raw);
    updateActive((tab) => {
      const nextHistory = [...tab.history.slice(0, tab.historyIndex + 1), url];
      return {
        ...tab,
        url,
        title: displayHost(url),
        history: nextHistory,
        historyIndex: nextHistory.length - 1,
      };
    });
    setSession((current) =>
      current ? { ...current, blocked: current.blocked + 3 + Math.floor(Math.random() * 8) } : current,
    );
  };

  const submitAddress = (event: FormEvent) => {
    event.preventDefault();
    navigate(address);
  };

  const moveHistory = (direction: -1 | 1) => {
    updateActive((tab) => {
      const index = tab.historyIndex + direction;
      if (index < 0 || index >= tab.history.length) return tab;
      const url = tab.history[index];
      return { ...tab, url, title: displayHost(url), historyIndex: index };
    });
  };

  const addTab = (isPrivate = false) => {
    const tab = createTab(isPrivate);
    setSession((current) =>
      current ? { ...current, tabs: [...current.tabs, tab], activeTabId: tab.id } : current,
    );
    setPanel(null);
  };

  const closeTab = (id: string) => {
    setSession((current) => {
      if (!current) return current;
      const remaining = current.tabs.filter((tab) => tab.id !== id);
      if (!remaining.length) return safeSession();
      return {
        ...current,
        tabs: remaining,
        activeTabId:
          current.activeTabId === id ? remaining[remaining.length - 1].id : current.activeTabId,
      };
    });
  };

  const toggleSetting = (key: keyof PrivacySettings) => {
    setSession((current) =>
      current
        ? { ...current, settings: { ...current.settings, [key]: !current.settings[key] } }
        : current,
    );
  };

  const clearSession = () => {
    const fresh = safeSession();
    sessionStorage.removeItem(SESSION_KEY);
    setSession(fresh);
    setPanel(null);
  };

  const sessionHistory = session.tabs.flatMap((tab) => tab.history).filter((url) => url !== "veil://start");

  return (
    <main className={activeTab.private ? "browser-shell private" : "browser-shell"}>
      <section className="browser-window" aria-label="Прототип браузера Veil">
        <header className="chrome">
          <div className="tabs-row">
            <div className="brand" aria-label="Veil Browser">
              <span className="brand-mark">V</span>
            </div>
            <div className="tabs" role="tablist" aria-label="Вкладки">
              {session.tabs.map((tab) => (
                <button
                  className={`tab ${tab.id === activeTab.id ? "active" : ""}`}
                  key={tab.id}
                  onClick={() => setSession({ ...session, activeTabId: tab.id })}
                  role="tab"
                  aria-selected={tab.id === activeTab.id}
                >
                  <span className="tab-icon">{tab.private ? "◒" : "●"}</span>
                  <span className="tab-title">{tab.title}</span>
                  <span
                    className="tab-close"
                    role="button"
                    aria-label={`Закрыть ${tab.title}`}
                    onClick={(event) => {
                      event.stopPropagation();
                      closeTab(tab.id);
                    }}
                  >
                    ×
                  </span>
                </button>
              ))}
              <button className="new-tab" onClick={() => addTab()} aria-label="Новая вкладка">+</button>
            </div>
            <div className="window-actions" aria-hidden="true"><span>—</span><span>□</span><span>×</span></div>
          </div>

          <div className="toolbar">
            <div className="nav-actions">
              <button onClick={() => moveHistory(-1)} disabled={activeTab.historyIndex === 0} aria-label="Назад">←</button>
              <button onClick={() => moveHistory(1)} disabled={activeTab.historyIndex >= activeTab.history.length - 1} aria-label="Вперёд">→</button>
              <button onClick={() => navigate(activeTab.url)} aria-label="Обновить">↻</button>
            </div>
            <form className="address-bar" onSubmit={submitAddress}>
              <span className="address-shield">◆</span>
              <input
                value={address}
                onChange={(event) => setAddress(event.target.value)}
                placeholder="Введите адрес или поисковый запрос"
                aria-label="Адрес или поисковый запрос"
              />
              <span className="onion-route" title="Защищённый маршрут">3 узла</span>
            </form>
            <button
              className="shield-button"
              onClick={() => setPanel(panel === "privacy" ? null : "privacy")}
              aria-label="Панель защиты"
            >
              <span>◆</span><b>{session.blocked}</b>
            </button>
            <button className="menu-button" onClick={() => setPanel(panel === "history" ? null : "history")} aria-label="Меню">•••</button>
          </div>
        </header>

        <section className="viewport">
          {activeTab.url === "veil://start" ? (
            <div className="start-page">
              <div className="session-pill"><i /> Временная сессия активна</div>
              <div className="hero-brand"><span className="hero-mark">V</span><h1>Veil</h1></div>
              <p className="hero-copy">Приватность — это стандарт,<br />а не дополнительная настройка.</p>
              <form className="hero-search" onSubmit={submitAddress}>
                <span>⌕</span>
                <input value={address} onChange={(event) => setAddress(event.target.value)} placeholder="Искать в интернете приватно" aria-label="Приватный поиск" />
                <kbd>Enter</kbd>
              </form>
              <div className="privacy-stats">
                <article><strong>{session.blocked}</strong><span>трекеров заблокировано</span></article>
                <article><strong>0</strong><span>данных сохранено</span></article>
                <article><strong>3</strong><span>защитных узла</span></article>
              </div>
              <div className="quick-actions">
                <button onClick={() => addTab(true)}><span>◒</span><b>Приватная вкладка</b><small>Отдельный контекст</small></button>
                <button onClick={() => setPanel("privacy")}><span>◆</span><b>Настроить защиту</b><small>4 уровня включено</small></button>
                <button onClick={() => setPanel("history")}><span>◷</span><b>История сессии</b><small>Исчезнет при закрытии</small></button>
              </div>
              <p className="prototype-note">UI-прототип · реальные сайты пока открываются в новой вкладке</p>
            </div>
          ) : (
            <div className="external-page">
              <div className="external-icon">↗</div>
              <span className="eyebrow">Внешний сайт</span>
              <h2>{displayHost(activeTab.url)}</h2>
              <p>Этот веб-прототип не может безопасно встроить страницу: сайты ограничивают загрузку через iframe.</p>
              <a href={activeTab.url} target="_blank" rel="noreferrer">Открыть сайт в новой вкладке</a>
              <button onClick={() => navigate("veil://start")}>Вернуться на стартовую</button>
            </div>
          )}

          {panel && (
            <aside className="side-panel" aria-label={panel === "privacy" ? "Настройки защиты" : "История сессии"}>
              <div className="panel-heading">
                <div><span>{panel === "privacy" ? "ЦЕНТР ЗАЩИТЫ" : "ТЕКУЩАЯ СЕССИЯ"}</span><h2>{panel === "privacy" ? "Защита активна" : "История"}</h2></div>
                <button onClick={() => setPanel(null)} aria-label="Закрыть панель">×</button>
              </div>
              {panel === "privacy" ? (
                <>
                  <div className="protection-score"><strong>94</strong><div><b>Высокий уровень</b><span>Демонстрационные настройки</span></div></div>
                  <div className="setting-list">
                    {([
                      ["trackers", "Блокировка трекеров", "Не даёт следить между сайтами"],
                      ["cookies", "Сторонние cookie", "Изолирует данные сайтов"],
                      ["fingerprint", "Защита отпечатка", "Снижает уникальность браузера"],
                      ["webrtc", "Защита WebRTC", "Скрывает локальные IP-адреса"],
                    ] as [keyof PrivacySettings, string, string][]).map(([key, title, note]) => (
                      <button className="setting" key={key} onClick={() => toggleSetting(key)}>
                        <span><b>{title}</b><small>{note}</small></span>
                        <i className={session.settings[key] ? "toggle on" : "toggle"}><em /></i>
                      </button>
                    ))}
                  </div>
                  <p className="honesty-note">Важно: это UI-прототип. Настоящая защита появится только в приложении на Chromium.</p>
                </>
              ) : (
                <>
                  <p className="history-note">Данные живут только до закрытия вкладки браузера.</p>
                  <div className="history-list">
                    {sessionHistory.length ? [...sessionHistory].reverse().map((url, index) => (
                      <button key={`${url}-${index}`} onClick={() => navigate(url)}><span>◉</span><div><b>{displayHost(url)}</b><small>{url}</small></div></button>
                    )) : <div className="empty-history">Здесь пока пусто</div>}
                  </div>
                  <button className="clear-session" onClick={clearSession}>Удалить данные сессии</button>
                </>
              )}
            </aside>
          )}
        </section>

        <footer className="statusbar">
          <span><i /> Защищено</span>
          <span>Сессия удалится при закрытии вкладки</span>
          <span>VEIL PROTOTYPE 0.1</span>
        </footer>
      </section>
    </main>
  );
}
