import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { App } from './App'

describe('App', () => {
  it('renders the login screen when unauthenticated', async () => {
    render(<App />)
    // With no token, the app redirects to /login which shows the panel title
    // and a Sign in button.
    expect(await screen.findByRole('button', { name: /sign in/i })).toBeInTheDocument()
    expect(screen.getByText('Xray Panel')).toBeInTheDocument()
  })
})
