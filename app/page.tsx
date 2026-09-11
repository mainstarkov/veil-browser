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
    <main
      className={activeTab.private ? "browser-shell private" : "browser-shell"}
      onPointerMove={(event) => {
        if (window.matchMedia("(pointer: coarse)").matches) return;
        const bounds = event.currentTarget.getBoundingClientRect();
        event.currentTarget.style.setProperty("--pointer-x", `${(event.clientX / bounds.width - 0.5) * 2}`);
        event.currentTarget.style.setProperty("--pointer-y", `${(event.clientY / bounds.height - 0.5) * 2}`);
      }}
    >
      <a className="skip-link" href="#browser-content">Перейти к браузеру</a>
      <section className="browser-window" aria-label="Браузер Veil">
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

        <section className="viewport" id="browser-content">
          <div className="cosmic-layer depth-0" aria-hidden="true">
            <div className="starfield" />
            <div className="horizon-grid" />
          </div>
          <div className="cosmic-layer depth-1" aria-hidden="true">
            <div className="aurora aurora-one" />
            <div className="aurora aurora-two" />
          </div>
          <div className="cosmic-layer depth-2" aria-hidden="true">
            <div className="orbit orbit-wide"><i /><i /><i /></div>
            <div className="signal-code signal-left">01001011</div>
            <div className="signal-code signal-right">NO TRACE / 03</div>
          </div>
          <div className="cosmic-layer depth-5" aria-hidden="true">
            <b className="particle particle-a" /><b className="particle particle-b" />
            <b className="particle particle-c" /><b className="particle particle-d" />
          </div>
          {activeTab.url === "veil://start" ? (
            <div className="start-page depth-4">
              <div className="session-pill"><i /> Следы исчезнут вместе с вкладкой</div>
              <div className="hero-brand"><span className="hero-mark">V</span><h1>Veil</h1><sup>01</sup></div>
              <form className="hero-search" onSubmit={submitAddress}>
                <span>⌕</span>
                <input value={address} onChange={(event) => setAddress(event.target.value)} placeholder="Куда идём?" aria-label="Приватный поиск" />
                <kbd>Enter</kbd>
              </form>
              <div className="quick-actions">
                <button onClick={() => addTab(true)}><span>◒</span><b>Уйти в инкогнито</b><small>Чистый контекст, без хвостов</small></button>
                <button onClick={() => setPanel("privacy")}><span>◆</span><b>Открыть щит</b><small>Посмотреть, что режем</small></button>
                <button onClick={() => setPanel("history")}><span>◷</span><b>Проверить следы</b><small>Только эта сессия</small></button>
              </div>
              <p className="prototype-note">Первый контур · движок на подходе</p>
            </div>
          ) : (
            <div className="external-page">
              <div className="external-icon">↗</div>
              <span className="eyebrow">Выход наружу</span>
              <h2>{displayHost(activeTab.url)}</h2>
              <p>Здесь тесно для чужого сайта. Пока движок не встроен, откроем его рядом — без фокусов с iframe.</p>
              <a href={activeTab.url} target="_blank" rel="noreferrer">Открыть рядом ↗</a>
              <button onClick={() => navigate("veil://start")}>Остаться в Veil</button>
            </div>
          )}

          {panel && (
            <aside className="side-panel" aria-label={panel === "privacy" ? "Настройки защиты" : "История сессии"}>
              <div className="panel-heading">
                <div><span>{panel === "privacy" ? "ЩИТ VEIL" : "СЛЕДЫ ЭТОЙ ВКЛАДКИ"}</span><h2>{panel === "privacy" ? "Здесь тихо" : "Куда заходили"}</h2></div>
                <button onClick={() => setPanel(null)} aria-label="Закрыть панель">×</button>
              </div>
              {panel === "privacy" ? (
                <>
                  <div className="protection-score"><strong>94</strong><div><b>Шума почти нет</b><span>4 фильтра держат линию</span></div></div>
                  <div className="setting-list">
                    {([
                      ["trackers", "Срезать маячки", "Рекламные сети теряют маршрут"],
                      ["cookies", "Не кормить cookie", "Каждый сайт сидит в своей клетке"],
                      ["fingerprint", "Смешаться с толпой", "Меньше уникальных признаков"],
                      ["webrtc", "Не светить локальный IP", "WebRTC держим под замком"],
                    ] as [keyof PrivacySettings, string, string][]).map(([key, title, note]) => (
                      <button className="setting" key={key} onClick={() => toggleSetting(key)}>
                        <span><b>{title}</b><small>{note}</small></span>
                        <i className={session.settings[key] ? "toggle on" : "toggle"}><em /></i>
                      </button>
                    ))}
                  </div>
                  <p className="honesty-note">Пока это интерфейсный прототип. Настоящие блокировки включим внутри Chromium — декорацию за защиту не выдаём.</p>
                </>
              ) : (
                <>
                  <p className="history-note">Обновить страницу можно. Закроете вкладку — список исчезнет.</p>
                  <div className="history-list">
                    {sessionHistory.length ? [...sessionHistory].reverse().map((url, index) => (
                      <button key={`${url}-${index}`} onClick={() => navigate(url)}><span>◉</span><div><b>{displayHost(url)}</b><small>{url}</small></div></button>
                    )) : <div className="empty-history">Чисто. Вы ещё никуда не ходили.</div>}
                  </div>
                  <button className="clear-session" onClick={clearSession}>Стереть всё сейчас</button>
                </>
              )}
            </aside>
          )}
        </section>

        <footer className="statusbar">
          <span><i /> Тихий режим</span>
          <span>Закроешь вкладку — сессия исчезнет</span>
          <span>VEIL / BUILD 01</span>
        </footer>
      </section>
    </main>
  );
}
