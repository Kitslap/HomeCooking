import { useState } from "react"
import { getToken, setToken, auth as authApi } from "@/lib/api"
import Layout from "@/components/Layout"
import Auth from "@/pages/Auth"
import Dashboard from "@/pages/Dashboard"
import Recipes from "@/pages/Recipes"
import Storage from "@/pages/Storage"

export type Page = "dashboard" | "recipes" | "storage"

export default function App() {
  const [authed, setAuthed] = useState(!!getToken())
  const [currentPage, setCurrentPage] = useState<Page>("dashboard")

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
