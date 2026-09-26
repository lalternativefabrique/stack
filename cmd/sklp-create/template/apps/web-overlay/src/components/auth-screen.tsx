type AuthScreenProps = {
  children: React.ReactNode
  footer?: React.ReactNode
  subtitle?: string
}

export function AuthScreen({ children, footer, subtitle }: AuthScreenProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-6 py-14 text-foreground">
      <div className="w-full max-w-sm">
        <div className="mb-10 flex flex-col items-center text-center">
          <h1 className="text-[1.75rem] leading-tight font-semibold tracking-tight">__APP_NAME__</h1>
          {subtitle && <p className="mt-2 text-sm text-muted-foreground">{subtitle}</p>}
        </div>
        {children}
        {footer && <p className="mt-6 text-center text-sm text-muted-foreground">{footer}</p>}
      </div>
    </div>
  )
}
