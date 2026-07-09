import { Loader2 } from 'lucide-react'

export default function AppLoading() {
  return (
    <div className="flex min-h-screen items-center justify-center p-6">
      <div className="text-center">
        <Loader2 size={34} className="mx-auto mb-3 animate-spin text-primary-600" />
        <p className="text-sm text-gray-500">Carregando informações...</p>
      </div>
    </div>
  )
}
