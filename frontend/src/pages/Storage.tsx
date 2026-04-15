import { useEffect, useState } from "react"
import { storage as api } from "@/lib/api"
import type { StorageItem, StorageInput } from "@/lib/api"

const CATEGORIES = ["Féculents", "Protéines", "Épicerie", "Frais", "Conserves", "Boissons", "Autre"]

const levelDot: Record<string, string> = {
  ok:       "#5a9e6f",
  low:      "#d4943a",
  critical: "#c85050",
}
const levelBadge: Record<string, { text: string; bg: string; border: string }> = {
  ok:       { text: "#5a9e6f", bg: "#5a9e6f14", border: "#5a9e6f30" },
  low:      { text: "#d4943a", bg: "#d4943a14", border: "#d4943a30" },
  critical: { text: "#c85050", bg: "#c8505014", border: "#c8505030" },
}

const EMPTY: StorageInput = { name: "", quantity: 0, unit: "pcs", category: "", alert_at: 0 }

const inp = "w-full rounded-lg px-3 py-2.5 text-sm outline-none transition-all"
const inpStyle = { background: "#1a1410", border: "1px solid #2e2418", color: "#f0e8dc" }

export default function Storage() {
  const [items, setItems]     = useState<StorageItem[]>([])
  const [filter, setFilter]   = useState("")
  const [search, setSearch]   = useState("")
  const [modal, setModal]     = useState<"create" | "edit" | null>(null)
  const [editing, setEditing] = useState<StorageItem | null>(null)
  const [form, setForm]       = useState<StorageInput>(EMPTY)
  const [saving, setSaving]   = useState(false)
  const [error, setError]     = useState("")

  const load = () =>
    api.list(filter || search ? { category: filter || undefined, search: search || undefined } : {})
      .then(r => setItems(r.data ?? [])).catch(() => {})

  useEffect(() => { load() }, [filter, search])

  const openCreate = () => { setForm(EMPTY); setEditing(null); setError(""); setModal("create") }
  const openEdit   = (item: StorageItem) => {
    setForm({ name: item.name, quantity: item.quantity, unit: item.unit, category: item.category ?? "", alert_at: item.alert_at, expiry: item.expiry, notes: item.notes })
    setEditing(item); setError(""); setModal("edit")
  }

  const save = async () => {
    setSaving(true); setError("")
    const payload = { ...form, unit: form.unit?.trim() || "pcs" }
    try {
      if (modal === "create") { await api.create(payload); await load() }
      else if (editing)       { await api.update(editing.id, payload); await load() }
      setModal(null)
    } catch (e: any) { setError(e.message) }
    finally { setSaving(false) }
  }

  const del = async (id: number) => {
    if (!confirm("Supprimer cet article ?")) return
    await api.delete(id).catch(() => {})
    setItems(l => l.filter(x => x.id !== id))
  }

  const adjust = async (item: StorageItem, delta: number) => {
    const updated = await api.adjust(item.id, delta).catch(() => null)
    if (updated) setItems(l => l.map(x => x.id === updated.id ? updated : x))
  }

  const f = (k: keyof StorageInput, v: any) => setForm(prev => ({ ...prev, [k]: v }))

  const alertCount = items.filter(i => i.level !== "ok").length

  return (
    <div className="space-y-5 max-w-5xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold" style={{ color: "#f0e8dc" }}>Inventaire</h1>
          <p className="text-sm mt-1" style={{ color: "#6a5040" }}>
            {alertCount > 0
              ? <span style={{ color: "#d4943a" }}>{alertCount} alerte{alertCount > 1 ? "s" : ""}</span>
              : <span style={{ color: "#5a9e6f" }}>✓ Tout est OK</span>}
            <span style={{ color: "#564a3a" }}> · {items.length} articles</span>
          </p>
        </div>
        <button onClick={openCreate}
          className="flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-semibold"
          style={{ background: "linear-gradient(135deg, #d4734a, #c05e38)", color: "#fff" }}>
          + Ajouter
        </button>
      </div>

      {/* Filtres : scroll horizontal sur mobile */}
      <div className="flex gap-2 overflow-x-auto pb-1 -mx-4 px-4 md:mx-0 md:px-0 md:flex-wrap">
        {["Tout", ...CATEGORIES].map(c => {
          const isActive = c === "Tout" ? !filter : filter === c
          return (
            <button key={c}
              onClick={() => setFilter(c === "Tout" ? "" : (filter === c ? "" : c))}
              className="px-3 py-1.5 rounded-lg text-xs transition-all flex-shrink-0"
              style={{
                background: isActive ? "#d4734a18" : "#141210",
                color:      isActive ? "#d4734a"   : "#8a7060",
                border:     isActive ? "1px solid #d4734a40" : "1px solid #2a2018",
              }}>
              {c}
            </button>
          )
        })}
        <input value={search} onChange={e => setSearch(e.target.value)} placeholder="Rechercher…"
          className="rounded-lg px-3 py-1.5 text-sm outline-none flex-shrink-0 w-36"
          style={{ ...inpStyle, marginLeft: "auto" }}
          onFocus={e => (e.target.style.borderColor = "#d4734a")}
          onBlur={e => (e.target.style.borderColor = "#2e2418")} />
      </div>

      {/* ── Vue tableau desktop (md+) ── */}
      <div className="hidden md:block rounded-xl overflow-hidden" style={{ background: "#141210", border: "1px solid #2a2018" }}>
        <div className="grid text-xs tracking-widests uppercase px-5 py-3"
          style={{ gridTemplateColumns: "1fr 130px 100px 120px 100px 90px", color: "#6a5040", borderBottom: "1px solid #2a2018" }}>
          <span>Produit</span><span>Quantité</span><span>Catégorie</span><span>Expiration</span><span>Niveau</span><span>Actions</span>
        </div>
        <div>
          {items.length === 0 && (
            <div className="py-12 text-center text-sm" style={{ color: "#564a3a" }}>
              Aucun article — cliquez sur + pour commencer
            </div>
          )}
          {items.map((item, idx) => {
            const badge = levelBadge[item.level] ?? levelBadge.ok
            return (
              <div key={item.id}
                className="grid items-center px-5 py-3 group transition-colors"
                style={{
                  gridTemplateColumns: "1fr 130px 100px 120px 100px 90px",
                  borderTop: idx > 0 ? "1px solid #1e1810" : undefined,
                  background: "transparent",
                }}
                onMouseEnter={e => (e.currentTarget.style.background = "#1a1610")}
                onMouseLeave={e => (e.currentTarget.style.background = "transparent")}>
                <div className="flex items-center gap-2.5">
                  <div className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: levelDot[item.level] ?? "#564a3a" }} />
                  <span className="text-sm" style={{ color: "#d8cfc4" }}>{item.name}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <button onClick={() => adjust(item, -1)}
                    className="w-6 h-6 rounded-md flex items-center justify-center text-sm transition-all"
                    style={{ background: "#1e1810", border: "1px solid #2a2018", color: "#8a7060" }}
                    onMouseEnter={e => { e.currentTarget.style.color = "#c85050"; e.currentTarget.style.borderColor = "#c8505040" }}
                    onMouseLeave={e => { e.currentTarget.style.color = "#8a7060"; e.currentTarget.style.borderColor = "#2a2018" }}>
                    −
                  </button>
                  <span className="text-sm tabular-nums min-w-[48px] text-center" style={{ color: "#d8cfc4" }}>
                    {item.quantity} <span style={{ color: "#8a7060", fontSize: "11px" }}>{item.unit}</span>
                  </span>
                  <button onClick={() => adjust(item, 1)}
                    className="w-6 h-6 rounded-md flex items-center justify-center text-sm transition-all"
                    style={{ background: "#1e1810", border: "1px solid #2a2018", color: "#8a7060" }}
                    onMouseEnter={e => { e.currentTarget.style.color = "#5a9e6f"; e.currentTarget.style.borderColor = "#5a9e6f40" }}
                    onMouseLeave={e => { e.currentTarget.style.color = "#8a7060"; e.currentTarget.style.borderColor = "#2a2018" }}>
                    +
                  </button>
                </div>
                <span className="text-sm" style={{ color: item.category ? "#8a7060" : "#3a3028" }}>{item.category ?? "—"}</span>
                <span className="text-sm" style={{ color: item.expiry ? "#8a7060" : "#3a3028" }}>{item.expiry ?? "—"}</span>
                <span className="text-xs px-2.5 py-1 rounded-full w-fit"
                  style={{ color: badge.text, background: badge.bg, border: `1px solid ${badge.border}` }}>
                  {item.level === "ok" ? "OK" : item.level === "low" ? "Faible" : "Critique"}
                </span>
                <div className="flex gap-3 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button onClick={() => openEdit(item)} className="text-xs" style={{ color: "#8a7060" }}
                    onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                    onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>Éditer</button>
                  <button onClick={() => del(item.id)} className="text-xs" style={{ color: "#8a7060" }}
                    onMouseEnter={e => (e.currentTarget.style.color = "#c85050")}
                    onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>Suppr.</button>
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* ── Vue cartes mobile (md-) ── */}
      <div className="md:hidden flex flex-col gap-3">
        {items.length === 0 && (
          <div className="py-12 text-center text-sm rounded-xl" style={{ color: "#564a3a", background: "#141210", border: "1px solid #2a2018" }}>
            Aucun article — cliquez sur + pour commencer
          </div>
        )}
        {items.map(item => {
          const badge = levelBadge[item.level] ?? levelBadge.ok
          return (
            <div key={item.id} className="p-4 rounded-xl" style={{ background: "#141210", border: "1px solid #2a2018" }}>
              {/* Ligne 1 : nom + badge */}
              <div className="flex items-center justify-between gap-3 mb-3">
                <div className="flex items-center gap-2.5 min-w-0">
                  <div className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: levelDot[item.level] ?? "#564a3a" }} />
                  <span className="text-sm font-medium truncate" style={{ color: "#d8cfc4" }}>{item.name}</span>
                </div>
                <span className="text-xs px-2.5 py-1 rounded-full flex-shrink-0"
                  style={{ color: badge.text, background: badge.bg, border: `1px solid ${badge.border}` }}>
                  {item.level === "ok" ? "OK" : item.level === "low" ? "Faible" : "Critique"}
                </span>
              </div>

              {/* Ligne 2 : quantité +/- + catégorie */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <button onClick={() => adjust(item, -1)}
                    className="w-8 h-8 rounded-lg flex items-center justify-center text-base transition-all"
                    style={{ background: "#1e1810", border: "1px solid #2a2018", color: "#8a7060" }}>
                    −
                  </button>
                  <span className="text-sm tabular-nums min-w-[56px] text-center" style={{ color: "#d8cfc4" }}>
                    {item.quantity} <span style={{ color: "#8a7060", fontSize: "11px" }}>{item.unit}</span>
                  </span>
                  <button onClick={() => adjust(item, 1)}
                    className="w-8 h-8 rounded-lg flex items-center justify-center text-base transition-all"
                    style={{ background: "#1e1810", border: "1px solid #2a2018", color: "#8a7060" }}>
                    +
                  </button>
                  {item.category && (
                    <span className="ml-2 text-xs px-2 py-0.5 rounded-full"
                      style={{ background: "#221a14", color: "#8a7060", border: "1px solid #2e2418" }}>
                      {item.category}
                    </span>
                  )}
                </div>

                {/* Actions */}
                <div className="flex gap-3">
                  <button onClick={() => openEdit(item)}
                    className="text-xs px-3 py-1.5 rounded-lg transition-all"
                    style={{ background: "#1e1a14", border: "1px solid #2e2418", color: "#d4734a" }}>
                    Éditer
                  </button>
                  <button onClick={() => del(item.id)}
                    className="text-xs px-3 py-1.5 rounded-lg transition-all"
                    style={{ background: "#1a1010", border: "1px solid #c8505030", color: "#c85050" }}>
                    ×
                  </button>
                </div>
              </div>

              {item.expiry && (
                <div className="mt-2 text-xs" style={{ color: "#6a5040" }}>Exp. {item.expiry}</div>
              )}
            </div>
          )
        })}
      </div>

      {/* Modal */}
      {modal && (
        <div className="fixed inset-0 flex items-end sm:items-center justify-center z-50 p-0 sm:p-4"
          style={{ background: "rgba(0,0,0,0.75)" }}>
          <div className="w-full sm:max-w-md max-h-[92vh] overflow-auto rounded-t-2xl sm:rounded-2xl"
            style={{ background: "#141210", border: "1px solid #2a2018" }}>

            <div className="relative flex items-center justify-between px-5 py-4" style={{ borderBottom: "1px solid #2a2018" }}>
              <div className="absolute top-2 left-1/2 -translate-x-1/2 w-10 h-1 rounded-full sm:hidden" style={{ background: "#2a2018" }} />
              <h2 className="text-base font-semibold" style={{ color: "#f0e8dc" }}>
                {modal === "create" ? "Nouvel article" : "Modifier l'article"}
              </h2>
              <button onClick={() => setModal(null)}
                className="w-8 h-8 flex items-center justify-center rounded-lg text-xl leading-none"
                style={{ color: "#564a3a", background: "#1e1810" }}>×</button>
            </div>

            <div className="p-5 space-y-4">
              <div>
                <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Nom *</label>
                <input value={form.name} onChange={e => f("name", e.target.value)}
                  className={inp} style={inpStyle}
                  onFocus={e => (e.target.style.borderColor = "#d4734a")}
                  onBlur={e => (e.target.style.borderColor = "#2e2418")} />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Quantité *</label>
                  <div className="flex items-center rounded-lg overflow-hidden" style={{ border: "1px solid #2e2418", background: "#1a1410" }}>
                    <button type="button" onClick={() => f("quantity", Math.max(0, (form.quantity ?? 0) - 1))}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>−</button>
                    <input type="number" min={0} step="1" value={form.quantity}
                      onChange={e => f("quantity", +e.target.value)}
                      className="flex-1 text-center py-2.5 text-sm outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                      style={{ background: "transparent", color: "#f0e8dc", border: "none" }}
                      onFocus={e => (e.currentTarget.parentElement!.style.borderColor = "#d4734a")}
                      onBlur={e => (e.currentTarget.parentElement!.style.borderColor = "#2e2418")} />
                    <button type="button" onClick={() => f("quantity", (form.quantity ?? 0) + 1)}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>+</button>
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Unité *</label>
                  <input value={form.unit} onChange={e => f("unit", e.target.value)}
                    placeholder="g, ml, pcs…"
                    className={inp} style={inpStyle}
                    onFocus={e => (e.target.style.borderColor = "#d4734a")}
                    onBlur={e => (e.target.style.borderColor = "#2e2418")} />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Catégorie</label>
                  <select value={form.category ?? ""} onChange={e => f("category", e.target.value)}
                    className={inp} style={{ ...inpStyle, cursor: "pointer" }}>
                    <option value="">—</option>
                    {CATEGORIES.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Seuil alerte</label>
                  <div className="flex items-center rounded-lg overflow-hidden" style={{ border: "1px solid #2e2418", background: "#1a1410" }}>
                    <button type="button" onClick={() => f("alert_at", Math.max(0, (form.alert_at ?? 0) - 1))}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>−</button>
                    <input type="number" min={0} step="1" value={form.alert_at ?? 0}
                      onChange={e => f("alert_at", +e.target.value)}
                      className="flex-1 text-center py-2.5 text-sm outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                      style={{ background: "transparent", color: "#f0e8dc", border: "none" }}
                      onFocus={e => (e.currentTarget.parentElement!.style.borderColor = "#d4734a")}
                      onBlur={e => (e.currentTarget.parentElement!.style.borderColor = "#2e2418")} />
                    <button type="button" onClick={() => f("alert_at", (form.alert_at ?? 0) + 1)}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>+</button>
                  </div>
                </div>
              </div>
              <div>
                <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Expiration</label>
                <input type="date" value={form.expiry ?? ""} onChange={e => f("expiry", e.target.value || undefined)}
                  className={inp} style={inpStyle}
                  onFocus={e => (e.target.style.borderColor = "#d4734a")}
                  onBlur={e => (e.target.style.borderColor = "#2e2418")} />
              </div>

              {error && (
                <div className="rounded-lg px-3 py-2.5 text-sm"
                  style={{ background: "#c8505018", border: "1px solid #c8505030", color: "#e07070" }}>
                  {error}
                </div>
              )}

              <div className="flex justify-end gap-3 pt-1 pb-2">
                <button onClick={() => setModal(null)}
                  className="px-5 py-2.5 rounded-lg text-sm"
                  style={{ background: "transparent", border: "1px solid #2a2018", color: "#8a7060" }}>
                  Annuler
                </button>
                <button onClick={save} disabled={saving || !form.name || !form.unit}
                  className="px-5 py-2.5 rounded-lg text-sm font-semibold disabled:opacity-50"
                  style={{ background: "linear-gradient(135deg, #d4734a, #c05e38)", color: "#fff" }}>
                  {saving ? "…" : "Enregistrer"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
