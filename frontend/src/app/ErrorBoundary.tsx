import { Component, type ErrorInfo, type ReactNode } from 'react'

type Props = {
  children: ReactNode
}

type State = {
  hasError: boolean
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Operator shell render error', error, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div
          style={{
            minHeight: '100dvh',
            display: 'grid',
            placeItems: 'center',
            background: '#080d17',
            color: '#e8f0ff',
            fontFamily: 'IBM Plex Sans, Segoe UI, sans-serif',
          }}
        >
          Виникла помилка рендерингу. Оновіть сторінку.
        </div>
      )
    }
    return this.props.children
  }
}

