import { useEffect, useState } from "react"
import { recipes as api } from "@/lib/api"
import type { Recipe, RecipeInput } from "@/lib/api"

const diffColor: Record<string, { text: string; bg: string; border: string }> = {
  facile:    { text: "#5a9e6f", bg: "#5a9e6f14", border: "#5a9e6f30" },
  moyen:     { text: "#d4943a", bg: "#d4943a14", border: "#d4943a30" },
  difficile: { text: "#c85050", bg: "#c8505014", border: "#c8505030" },
}

const EMPTY: RecipeInput = {
  name: "", description: "", servings: 4, difficulty: "facile", tags: [],
  ingredients: [{ name: "", unit: "pcs" }], steps: [{ step_order: 1, content: "" }],
}

const inp = "w-full rounded-lg px-3 py-2.5 text-sm outline-none transition-all"
const inpStyle = { background: "#1a1410", border: "1px solid #2e2418", color: "#f0e8dc" }

export default function Recipes() {
  const [list, setList]         = useState<Recipe[]>([])
  const [selected, setSelected] = useState<Recipe | null>(null)
  const [search, setSearch]     = useState("")
  const [modal, setModal]       = useState<"create" | "edit" | null>(null)
  const [form, setForm]         = useState<RecipeInput>(EMPTY)
  const [saving, setSaving]     = useState(false)
  const [error, setError]       = useState("")
  const [tagInput, setTagInput] = useState("")
  // Vue mobile : "list" ou "detail"
  const [mobileView, setMobileView] = useState<"list" | "detail">("list")

  const load = (q?: string) =>
    api.list(q ? { search: q } : {}).then(r => setList(r.data ?? [])).catch(() => {})

  useEffect(() => { load() }, [])

  const openCreate = () => { setForm(EMPTY); setTagInput(""); setError(""); setModal("create") }
  const openEdit   = (r: Recipe) => {
    setForm({
      name: r.name, description: r.description ?? "", servings: r.servings,
      difficulty: r.difficulty, tags: r.tags ?? [],
      ingredients: r.ingredients?.map(i => ({ name: i.name, quantity: i.quantity, unit: i.unit ?? "pcs" })) ?? [],
      steps: r.steps?.map(s => ({ step_order: s.step_order, content: s.content })) ?? [],
    })
    setTagInput((r.tags ?? []).join(", "))
    setError(""); setModal("edit")
  }

  const selectRecipe = async (r: Recipe) => {
    const full = await api.get(r.id)
    setSelected(full)
    setMobileView("detail")
  }

  const save = async () => {
    setSaving(true); setError("")
    const payload = {
      ...form,
      ingredients: form.ingredients?.map(i => ({ ...i, unit: i.unit?.trim() || "pcs" })),
    }
    try {
      if (modal === "create") {
        const r = await api.create(payload); setList(l => [r, ...l]); setSelected(r); setMobileView("detail")
      } else if (modal === "edit" && selected) {
        const r = await api.update(selected.id, payload)
        setList(l => l.map(x => x.id === r.id ? r : x)); setSelected(r)
      }
      setModal(null)
    } catch (e: any) { setError(e.message) }
    finally { setSaving(false) }
  }

  const del = async (id: number) => {
    if (!confirm("Supprimer cette recette ?")) return
    await api.delete(id).catch(() => {})
    setList(l => l.filter(x => x.id !== id))
    if (selected?.id === id) { setSelected(null); setMobileView("list") }
  }

  const setIng  = (i: number, field: string, val: any) =>
    setForm(f => ({ ...f, ingredients: f.ingredients!.map((x, j) => j === i ? { ...x, [field]: val } : x) }))
  const setStep = (i: number, val: string) =>
    setForm(f => ({ ...f, steps: f.steps!.map((x, j) => j === i ? { ...x, content: val } : x) }))
  const addIng  = () => setForm(f => ({ ...f, ingredients: [...f.ingredients!, { name: "", unit: "pcs" }] }))
  const addStep = () => setForm(f => ({ ...f, steps: [...f.steps!, { step_order: f.steps!.length + 1, content: "" }] }))
  const delIng  = (i: number) => setForm(f => ({ ...f, ingredients: f.ingredients!.filter((_, j) => j !== i) }))
  const delStep = (i: number) => setForm(f => ({ ...f, steps: f.steps!.filter((_, j) => j !== i) }))

  /* ── Panneau liste ── */
  const ListPanel = () => (
    <div className="flex flex-col gap-3 h-full">
      <div className="flex gap-2">
        <input value={search} onChange={e => { setSearch(e.target.value); load(e.target.value) }}
          placeholder="Rechercher une recette…"
          className="flex-1 rounded-lg px-3 py-2 text-sm outline-none"
          style={inpStyle}
          onFocus={e => (e.target.style.borderColor = "#d4734a")}
          onBlur={e => (e.target.style.borderColor = "#2e2418")} />
        <button onClick={openCreate}
          className="px-4 py-2 rounded-lg text-sm font-semibold"
          style={{ background: "linear-gradient(135deg, #d4734a, #c05e38)", color: "#fff" }}>
          +
        </button>
      </div>
      <div className="text-xs tracking-widest uppercase px-1" style={{ color: "#6a5040" }}>
        {list.length} recette{list.length > 1 ? "s" : ""}
      </div>
      <div className="flex flex-col gap-1.5 overflow-auto flex-1">
        {list.length === 0 && (
          <p className="text-sm text-center mt-8" style={{ color: "#564a3a" }}>
            Aucune recette.<br />Cliquez sur + pour commencer.
          </p>
        )}
        {list.map(r => {
          const diff = diffColor[r.difficulty ?? ""] ?? null
          const isActive = selected?.id === r.id
          return (
            <button key={r.id} onClick={() => selectRecipe(r)}
              className="flex flex-col gap-2 p-3.5 rounded-xl text-left transition-all"
              style={{
                background: isActive ? "#1e1610" : "#141210",
                border: isActive ? "1px solid #d4734a40" : "1px solid #2a2018",
              }}>
              <div className="flex items-start justify-between gap-2">
                <span className="text-sm font-medium" style={{ color: "#d8cfc4" }}>{r.name}</span>
                {diff && (
                  <span className="text-xs px-2 py-0.5 rounded-full flex-shrink-0"
                    style={{ color: diff.text, background: diff.bg, border: `1px solid ${diff.border}` }}>
                    {r.difficulty}
                  </span>
                )}
              </div>
              {r.tags && r.tags.length > 0 && (
                <div className="flex gap-1 flex-wrap">
                  {r.tags.slice(0, 3).map(t => (
                    <span key={t} className="text-xs px-2 py-0.5 rounded-full"
                      style={{ background: "#221a14", color: "#8a7060", border: "1px solid #2e2418" }}>
                      {t}
                    </span>
                  ))}
                </div>
              )}
            </button>
          )
        })}
      </div>
    </div>
  )

  /* ── Panneau détail ── */
  const DetailPanel = () => (
    <div className="flex-1 rounded-xl overflow-auto h-full" style={{ background: "#141210", border: "1px solid #2a2018" }}>
      {selected ? (
        <div className="p-5 md:p-6">
          {/* Bouton retour mobile */}
          <button onClick={() => setMobileView("list")}
            className="md:hidden flex items-center gap-1.5 text-sm mb-4 transition-colors"
            style={{ color: "#8a7060" }}
            onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
            onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>
            ← Retour
          </button>

          <div className="flex items-start justify-between mb-5 gap-3">
            <div className="min-w-0">
              <h2 className="text-lg font-semibold" style={{ color: "#f0e8dc" }}>{selected.name}</h2>
              <div className="flex flex-wrap items-center gap-3 mt-1.5 text-xs" style={{ color: "#8a7060" }}>
                {selected.prep_time && <span>⏱ {selected.prep_time} min prép.</span>}
                {selected.cook_time && <span>⏱ {selected.cook_time} min cuisson</span>}
                <span>◯ {selected.servings} pers.</span>
              </div>
            </div>
            <div className="flex gap-2 flex-shrink-0">
              <button onClick={() => openEdit(selected)}
                className="text-xs px-3 py-2 rounded-lg transition-all"
                style={{ background: "#1e1a14", border: "1px solid #2e2418", color: "#d4734a" }}>
                Modifier
              </button>
              <button onClick={() => del(selected.id)}
                className="text-xs px-3 py-2 rounded-lg transition-all"
                style={{ background: "#1a1010", border: "1px solid #c8505030", color: "#c85050" }}>
                Suppr.
              </button>
            </div>
          </div>

          {selected.description && (
            <p className="text-sm mb-5 leading-relaxed" style={{ color: "#8a7060" }}>{selected.description}</p>
          )}

          {/* Ingrédients + Étapes : 1 col mobile → 2 cols desktop */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h3 className="text-xs tracking-widests uppercase mb-3" style={{ color: "#6a5040" }}>Ingrédients</h3>
              {selected.ingredients?.map(i => (
                <div key={i.id} className="flex gap-2 text-sm mb-1.5" style={{ color: "#d8cfc4" }}>
                  <span style={{ color: "#d4734a" }}>—</span>
                  {i.quantity ? `${i.quantity} ${i.unit} ` : ""}{i.name}
                </div>
              ))}
            </div>
            <div>
              <h3 className="text-xs tracking-widests uppercase mb-3" style={{ color: "#6a5040" }}>Étapes</h3>
              {selected.steps?.map(s => (
                <div key={s.id} className="flex gap-3 mb-3">
                  <span className="text-xs font-mono flex-shrink-0 mt-0.5 w-5" style={{ color: "#d4734a" }}>
                    {String(s.step_order).padStart(2, "0")}
                  </span>
                  <span className="text-sm leading-relaxed" style={{ color: "#d8cfc4" }}>{s.content}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      ) : (
        <div className="h-full flex flex-col items-center justify-center gap-3" style={{ color: "#4a3828" }}>
          <span className="text-5xl opacity-50">◈</span>
          <span className="text-sm">Sélectionnez ou créez une recette</span>
        </div>
      )}
    </div>
  )

  return (
    <>
      {/* ── Desktop : split panel ── */}
      <div className="hidden md:flex gap-4 max-w-6xl" style={{ height: "calc(100vh - 80px)" }}>
        <div className="w-80 flex-shrink-0 flex flex-col" style={{ height: "100%" }}>
          <ListPanel />
        </div>
        <DetailPanel />
      </div>

      {/* ── Mobile : vue toggle liste / détail ── */}
      <div className="md:hidden" style={{ minHeight: "calc(100vh - 140px)" }}>
        {mobileView === "list" ? <ListPanel /> : <DetailPanel />}
      </div>

      {/* ── Modal create/edit ── */}
      {modal && (
        <div className="fixed inset-0 flex items-end sm:items-center justify-center z-50 p-0 sm:p-4"
          style={{ background: "rgba(0,0,0,0.75)" }}>
          {/* Sur mobile : sheet qui monte du bas ; sur sm+ : dialog centré */}
          <div className="w-full sm:max-w-2xl max-h-[92vh] overflow-auto rounded-t-2xl sm:rounded-2xl"
            style={{ background: "#141210", border: "1px solid #2a2018" }}>

            <div className="flex items-center justify-between px-5 py-4" style={{ borderBottom: "1px solid #2a2018" }}>
              {/* Barre de tirage mobile */}
              <div className="absolute top-2 left-1/2 -translate-x-1/2 w-10 h-1 rounded-full sm:hidden" style={{ background: "#2a2018" }} />
              <h2 className="text-base font-semibold" style={{ color: "#f0e8dc" }}>
                {modal === "create" ? "Nouvelle recette" : "Modifier la recette"}
              </h2>
              <button onClick={() => setModal(null)}
                className="text-xl leading-none w-8 h-8 flex items-center justify-center rounded-lg transition-colors"
                style={{ color: "#564a3a", background: "#1e1810" }}>
                ×
              </button>
            </div>

            <div className="p-5 space-y-5">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Nom *</label>
                  <input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                    className={inp} style={inpStyle}
                    onFocus={e => (e.target.style.borderColor = "#d4734a")}
                    onBlur={e => (e.target.style.borderColor = "#2e2418")} />
                </div>
                <div>
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Portions</label>
                  <div className="flex items-center rounded-lg overflow-hidden" style={{ border: "1px solid #2e2418", background: "#1a1410" }}>
                    <button type="button" onClick={() => setForm(f => ({ ...f, servings: Math.max(1, (f.servings ?? 1) - 1) }))}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>−</button>
                    <input type="number" min={1} step="1" value={form.servings}
                      onChange={e => setForm(f => ({ ...f, servings: +e.target.value }))}
                      className="flex-1 min-w-0 text-center py-2.5 text-sm outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                      style={{ background: "transparent", color: "#f0e8dc", border: "none" }}
                      onFocus={e => (e.currentTarget.parentElement!.style.borderColor = "#d4734a")}
                      onBlur={e => (e.currentTarget.parentElement!.style.borderColor = "#2e2418")} />
                    <button type="button" onClick={() => setForm(f => ({ ...f, servings: (f.servings ?? 1) + 1 }))}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>+</button>
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Difficulté</label>
                  <select value={form.difficulty ?? ""} onChange={e => setForm(f => ({ ...f, difficulty: e.target.value as any }))}
                    className={inp} style={{ ...inpStyle, cursor: "pointer" }}>
                    <option value="facile">Facile</option>
                    <option value="moyen">Moyen</option>
                    <option value="difficile">Difficile</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Prép. (min)</label>
                  <div className="flex items-center rounded-lg overflow-hidden" style={{ border: "1px solid #2e2418", background: "#1a1410" }}>
                    <button type="button" onClick={() => setForm(f => ({ ...f, prep_time: Math.max(0, (f.prep_time ?? 0) - 1) || undefined }))}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>−</button>
                    <input type="number" min={0} step="1" value={form.prep_time ?? ""}
                      onChange={e => setForm(f => ({ ...f, prep_time: +e.target.value || undefined }))}
                      className="flex-1 min-w-0 text-center py-2.5 text-sm outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                      style={{ background: "transparent", color: "#f0e8dc", border: "none" }}
                      onFocus={e => (e.currentTarget.parentElement!.style.borderColor = "#d4734a")}
                      onBlur={e => (e.currentTarget.parentElement!.style.borderColor = "#2e2418")} />
                    <button type="button" onClick={() => setForm(f => ({ ...f, prep_time: (f.prep_time ?? 0) + 1 }))}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>+</button>
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Cuisson (min)</label>
                  <div className="flex items-center rounded-lg overflow-hidden" style={{ border: "1px solid #2e2418", background: "#1a1410" }}>
                    <button type="button" onClick={() => setForm(f => ({ ...f, cook_time: Math.max(0, (f.cook_time ?? 0) - 1) || undefined }))}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>−</button>
                    <input type="number" min={0} step="1" value={form.cook_time ?? ""}
                      onChange={e => setForm(f => ({ ...f, cook_time: +e.target.value || undefined }))}
                      className="flex-1 min-w-0 text-center py-2.5 text-sm outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                      style={{ background: "transparent", color: "#f0e8dc", border: "none" }}
                      onFocus={e => (e.currentTarget.parentElement!.style.borderColor = "#d4734a")}
                      onBlur={e => (e.currentTarget.parentElement!.style.borderColor = "#2e2418")} />
                    <button type="button" onClick={() => setForm(f => ({ ...f, cook_time: (f.cook_time ?? 0) + 1 }))}
                      className="w-9 h-10 flex items-center justify-center text-sm transition-colors shrink-0"
                      style={{ color: "#8a7060" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>+</button>
                  </div>
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium mb-1.5 tracking-widests uppercase" style={{ color: "#8a7060" }}>Tags (séparés par virgule)</label>
                  <input
                    value={tagInput}
                    onChange={e => setTagInput(e.target.value)}
                    placeholder="végé, riz, hivernal"
                    className={inp} style={inpStyle}
                    onFocus={e => (e.target.style.borderColor = "#d4734a")}
                    onBlur={e => {
                      e.target.style.borderColor = "#2e2418"
                      setForm(f => ({ ...f, tags: tagInput.split(",").map(t => t.trim()).filter(Boolean) }))
                    }} />
                </div>
              </div>

              {/* Ingrédients */}
              <div>
                <div className="flex items-center justify-between mb-3">
                  <label className="text-xs tracking-widests uppercase font-medium" style={{ color: "#8a7060" }}>Ingrédients</label>
                  <button onClick={addIng} className="text-xs" style={{ color: "#d4734a" }}>+ Ajouter</button>
                </div>
                {form.ingredients?.map((ing, i) => (
                  <div key={i} className="flex gap-2 mb-2">
                    <div className="flex items-center rounded-lg overflow-hidden" style={{ border: "1px solid #2e2418", background: "#1a1410" }}>
                      <button type="button" onClick={() => setIng(i, "quantity", Math.max(0, (ing.quantity ?? 0) - 1))}
                        className="w-7 h-9 flex items-center justify-center text-sm transition-colors shrink-0"
                        style={{ color: "#8a7060" }}
                        onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                        onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>−</button>
                      <input value={ing.quantity ?? ""} onChange={e => setIng(i, "quantity", +e.target.value || undefined)}
                        placeholder="Qté" type="number" min={0}
                        className="w-10 text-center py-2 text-sm outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                        style={{ background: "transparent", color: "#f0e8dc", border: "none" }}
                        onFocus={e => (e.currentTarget.parentElement!.style.borderColor = "#d4734a")}
                        onBlur={e => (e.currentTarget.parentElement!.style.borderColor = "#2e2418")} />
                      <button type="button" onClick={() => setIng(i, "quantity", (ing.quantity ?? 0) + 1)}
                        className="w-7 h-9 flex items-center justify-center text-sm transition-colors shrink-0"
                        style={{ color: "#8a7060" }}
                        onMouseEnter={e => (e.currentTarget.style.color = "#d4734a")}
                        onMouseLeave={e => (e.currentTarget.style.color = "#8a7060")}>+</button>
                    </div>
                    <input value={ing.unit ?? ""} onChange={e => setIng(i, "unit", e.target.value)}
                      placeholder="Unité"
                      className="w-16 rounded-lg px-2 py-2 text-sm outline-none" style={inpStyle}
                      onFocus={e => (e.target.style.borderColor = "#d4734a")}
                      onBlur={e => (e.target.style.borderColor = "#2e2418")} />
                    <input value={ing.name} onChange={e => setIng(i, "name", e.target.value)}
                      placeholder="Nom de l'ingrédient" required
                      className="flex-1 rounded-lg px-2 py-2 text-sm outline-none" style={inpStyle}
                      onFocus={e => (e.target.style.borderColor = "#d4734a")}
                      onBlur={e => (e.target.style.borderColor = "#2e2418")} />
                    <button onClick={() => delIng(i)}
                      className="text-lg leading-none self-center transition-colors"
                      style={{ color: "#564a3a" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#c85050")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#564a3a")}>×</button>
                  </div>
                ))}
              </div>

              {/* Étapes */}
              <div>
                <div className="flex items-center justify-between mb-3">
                  <label className="text-xs tracking-widests uppercase font-medium" style={{ color: "#8a7060" }}>Étapes</label>
                  <button onClick={addStep} className="text-xs" style={{ color: "#d4734a" }}>+ Ajouter</button>
                </div>
                {form.steps?.map((s, i) => (
                  <div key={i} className="flex gap-3 mb-2">
                    <span className="text-xs font-mono mt-2.5 w-5 flex-shrink-0" style={{ color: "#d4734a" }}>
                      {String(i + 1).padStart(2, "0")}
                    </span>
                    <textarea value={s.content} onChange={e => setStep(i, e.target.value)} rows={2}
                      placeholder="Décrivez cette étape…"
                      className="flex-1 rounded-lg px-3 py-2 text-sm outline-none resize-none" style={inpStyle}
                      onFocus={e => (e.target.style.borderColor = "#d4734a")}
                      onBlur={e => (e.target.style.borderColor = "#2e2418")} />
                    <button onClick={() => delStep(i)}
                      className="text-lg leading-none self-start mt-1 transition-colors"
                      style={{ color: "#564a3a" }}
                      onMouseEnter={e => (e.currentTarget.style.color = "#c85050")}
                      onMouseLeave={e => (e.currentTarget.style.color = "#564a3a")}>×</button>
                  </div>
                ))}
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
                <button onClick={save} disabled={saving || !form.name}
                  className="px-5 py-2.5 rounded-lg text-sm font-semibold disabled:opacity-50"
                  style={{ background: "linear-gradient(135deg, #d4734a, #c05e38)", color: "#fff" }}>
                  {saving ? "Enregistrement…" : "Enregistrer"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
