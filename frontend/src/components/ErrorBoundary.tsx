import React, { Component, type ReactNode } from "react"
import { WarningCircle, ArrowClockwise } from "@phosphor-icons/react"
import { Button } from "@/components/ui/button"

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("Uncaught application error:", error, errorInfo)
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null })
    window.location.reload()
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex min-h-[60vh] flex-col items-center justify-center p-6 text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-destructive/10 text-destructive border border-destructive/20 shadow-lg">
            <WarningCircle className="h-7 w-7" weight="fill" />
          </div>
          <h2 className="font-display text-xl font-bold text-foreground mb-2">
            Something went wrong
          </h2>
          <p className="text-xs text-muted-foreground max-w-md mb-6 leading-relaxed">
            {this.state.error?.message || "An unexpected error occurred while rendering this component."}
          </p>
          <Button
            onClick={this.handleReset}
            variant="outline"
            size="sm"
            className="gap-2 text-xs font-semibold"
          >
            <ArrowClockwise className="h-3.5 w-3.5" />
            Reload Workspace
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}
