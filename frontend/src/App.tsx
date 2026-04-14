import { useState, useEffect } from "react"
import { getToken, setToken, auth as authApi, setupApi } from "@/lib/api"
import Layout from "@/components/Layout"
import Auth from "@/pages/Auth"
import Setup from "@/pages/Setup"
import Dashboard from "@/pages/Dashboard"
import Recipes from "@/pages/Recipes"
import Storage from "@/pages/Storage"
import logoIcon from "@/assets/home_cooking_appicon.png"

export type Page = "dashboard" | "recipes" | "storage"

export default function App() {
  const [authed, setAuthed] = useState(!!getToken())
  const [currentPage, setCurrentPage] = useState<Page>("dashboard")
  // null = chargement en cours, true = setup nécessaire, false = déjà configuré
  const [setupNeeded, setSetupNeeded] = useState<boolean | null>(null)

  // Vérifie au montage si l'instance a besoin du setup initial
  useEffect(() => {
    setupApi.status()
      .then(res => setSetupNeeded(res.needs_setup))
      .catch(() => setSetupNeeded(false)) // en cas d'erreur, on assume configuré
  }, [])

  // Écran de chargement pendant la vérification du statut
  if (setupNeeded === null) {
    return (
      <div className="flex h-screen items-center justify-center" style={{ background: "#0e0c0b" }}>
        <img src={logoIcon} alt="Home Cooking" className="w-12 h-12 animate-pulse" />
      </div>
    )
  }

  // Setup wizard si l'instance n'a jamais été configurée
  if (setupNeeded) {
    return <Setup onComplete={() => { setSetupNeeded(false); setAuthed(true) }} />
  }

  // Écran de login si l'utilisateur n'est pas authentifié
  if (!authed) return <Auth onLogin={() => setAuthed(true)} />

  const handleLogout = async () => {
    await authApi.logout().catch(() => {})
    setToken(null)
    setAuthed(false)
  }

  const renderPage = () => {
    switch (currentPage) {
      case "dashboard": return <Dashboard onNavigate={setCurrentPage} />
      case "recipes":   return <Recipes />
      case "storage":   return <Storage />
    }
  }

  return (
    <Layout currentPage={currentPage} onNavigate={setCurrentPage} onLogout={handleLogout}>
      {renderPage()}
    </Layout>
  )
}
