import { useState } from "react"
import type { ReactNode } from "react"
import type { Page } from "@/App"
import logoIcon from "@/assets/home_cooking_icon.png"

interface LayoutProps {
  children: ReactNode
  currentPage: Page
  onNavigate: (page: Page) => void
  onLogout: () => void
}

const navItems: { id: Page; label: string; icon: string }[] = [
  { id: "dashboard", label: "Accueil",    icon: "⌂" },
  { id: "recipes",   label: "Recettes",   icon: "◈" },
  { id: "storage",   label: "Inventaire", icon: "▣" },
]

export default function Layout({ children, currentPage, onNavigate, onLogout }: LayoutProps) {
  const [collapsed, setCollapsed] = useState(false)

  return (
    <div className="flex h-screen overflow-hidden" style={{ background: "#0e0c0b", color: "#f0e8dc" }}>

      {/* ── Sidebar desktop (md+) ── */}
      <aside
        className={`hidden md:flex flex-col flex-shrink-0 transition-all duration-200 ${collapsed ? "w-14" : "w-52"}`}
        style={{ background: "#120f0d", borderRight: "1px solid #2a2018" }}>

        {/* Logo */}
        <div className="flex items-center gap-3 h-14 px-3 flex-shrink-0" style={{ borderBottom: "1px solid #2a2018" }}>
          <img src={logoIcon} alt="Home Cooking" className="w-8 h-8 rounded flex-shrink-0" />
          {!collapsed && (
            <div>
              <div className="text-sm font-semibold tracking-wide" style={{ color: "#f0e8dc" }}>Home Cooking</div>
              <div className="text-xs" style={{ color: "#6a5040", fontSize: "10px" }}>Ma cuisine</div>
            </div>
          )}
        </div>

        {/* Nav */}
        <nav className="flex-1 py-3 px-2 flex flex-col gap-0.5">
          {navItems.map(item => {
            const active = currentPage === item.id
            return (
              <button key={item.id} onClick={() => onNavigate(item.id)}
                title={collapsed ? item.label : undefined}
                className="flex items-center gap-3 rounded px-2.5 py-2.5 text-left w-full transition-all duration-150"
                style={{
                  background: active ? "#d4734a18" : "transparent",
                  color: active ? "#d4734a" : "#8a7060",
                  fontWeight: active ? 600 : 400,
                }}>
                <span className="flex-shrink-0 text-base w-5 text-center">{item.icon}</span>
                {!collapsed && <span className="text-sm truncate">{item.label}</span>}
                {active && !collapsed && (
                  <div className="ml-auto w-1 h-4 rounded-full" style={{ background: "#d4734a" }} />
                )}
              </button>
            )
          })}
        </nav>

        {/* Collapse toggle */}
        <div className="p-2 flex-shrink-0" style={{ borderTop: "1px solid #2a2018" }}>
          <button onClick={() => setCollapsed(!collapsed)}
            className="w-full flex items-center justify-center gap-2 py-2 px-2 rounded transition-colors text-xs"
            style={{ color: "#564a3a" }}
            onMouseEnter={e => (e.currentTarget.style.color = "#8a7060")}
            onMouseLeave={e => (e.currentTarget.style.color = "#564a3a")}>
            <span className="transition-transform duration-200" style={{ transform: collapsed ? "rotate(180deg)" : "none" }}>◂</span>
            {!collapsed && <span style={{ fontSize: "11px" }}>Réduire</span>}
          </button>
        </div>
      </aside>

      {/* ── Main ── */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">

        {/* Topbar */}
        <header className="h-14 flex items-center justify-between px-4 md:px-6 flex-shrink-0"
          style={{ borderBottom: "1px solid #2a2018", background: "#0e0c0b" }}>

          {/* Mobile : logo */}
          <div className="flex items-center gap-2 md:hidden">
            <img src={logoIcon} alt="Home Cooking" className="w-7 h-7 rounded" />
            <span className="text-sm font-semibold" style={{ color: "#f0e8dc" }}>Home Cooking</span>
          </div>

          {/* Desktop : titre page */}
          <span className="hidden md:block text-sm font-medium" style={{ color: "#8a7060" }}>
            {navItems.find(n => n.id === currentPage)?.label}
          </span>

          <div className="flex items-center gap-3">
            <div className="hidden sm:flex items-center gap-1.5">
              <div className="w-1.5 h-1.5 rounded-full animate-pulse" style={{ background: "#5a9e6f" }} />
              <span style={{ fontSize: "11px", color: "#564a3a" }}>v0.1</span>
            </div>
            <button onClick={onLogout}
              className="w-8 h-8 rounded flex items-center justify-center text-xs font-semibold transition-all"
              style={{ background: "#1e1810", border: "1px solid #2a2018", color: "#8a7060" }}
              onMouseEnter={e => { e.currentTarget.style.borderColor = "#c85050"; e.currentTarget.style.color = "#c85050" }}
              onMouseLeave={e => { e.currentTarget.style.borderColor = "#2a2018"; e.currentTarget.style.color = "#8a7060" }}
              title="Déconnexion">
              ⏻
            </button>
          </div>
        </header>

        {/* Content — padding bottom sur mobile pour la bottom nav */}
        <main className="flex-1 overflow-auto p-4 md:p-6 pb-20 md:pb-6">
          {children}
        </main>
      </div>

      {/* ── Bottom nav mobile (md-) ── */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 flex z-40"
        style={{ background: "#120f0d", borderTop: "1px solid #2a2018" }}>
        {navItems.map(item => {
          const active = currentPage === item.id
          return (
            <button key={item.id} onClick={() => onNavigate(item.id)}
              className="flex-1 flex flex-col items-center justify-center gap-1 py-3 transition-all"
              style={{ color: active ? "#d4734a" : "#564a3a" }}>
              <span className="text-lg leading-none">{item.icon}</span>
              <span style={{ fontSize: "10px", fontWeight: active ? 600 : 400 }}>{item.label}</span>
              {active && (
                <div className="absolute top-0 w-8 h-0.5 rounded-full" style={{ background: "#d4734a" }} />
              )}
            </button>
          )
        })}
      </nav>
    </div>
  )
}
