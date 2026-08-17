import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { mockNavigate } from '@/test/setup'
import LoginPage from '../LoginPage'
import { useAuthStore } from '@/stores/auth'

function renderLogin() {
  return render(
    <MemoryRouter>
      <LoginPage />
    </MemoryRouter>,
  )
}

describe('LoginPage', () => {
  beforeEach(() => {
    mockNavigate.mockClear()
    // Reset auth store between tests
    useAuthStore.setState({
      user: null,
      accessToken: null,
      refreshTokenValue: null,
      isAuthenticated: false,
      isLoading: false,
      pendingToken: null,
    })
  })

  it('renders email and password fields', () => {
    renderLogin()
    expect(screen.getByLabelText('auth.email')).toBeInTheDocument()
    expect(screen.getByLabelText('auth.password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'auth.login' })).toBeInTheDocument()
  })

  it('logs in with valid credentials and navigates to /', async () => {
    // Mock the login method to succeed
    const loginMock = vi.fn().mockResolvedValue(undefined)
    useAuthStore.setState({ login: loginMock })

    const user = userEvent.setup()
    renderLogin()

    await user.type(screen.getByLabelText('auth.email'), 'test@firma.de')
    await user.type(screen.getByLabelText('auth.password'), 'correct')
    await user.click(screen.getByRole('button', { name: 'auth.login' }))

    await waitFor(() => {
      expect(loginMock).toHaveBeenCalledWith('test@firma.de', 'correct', true)
    })
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true })
    })
  })

  // The store throws i18n keys, so the page must translate them. The test's t()
  // mock returns the key verbatim, which is what these assertions match on.
  it.each([
    ['auth.invalidCredentials', 'wrong password'],
    ['auth.rateLimited', 'too many attempts'],
    ['auth.serverUnreachable', 'backend down'],
  ])('renders %s as its own message', async (key) => {
    const loginMock = vi.fn().mockRejectedValue(new Error(key))
    useAuthStore.setState({ login: loginMock })

    const user = userEvent.setup()
    renderLogin()

    await user.type(screen.getByLabelText('auth.email'), 'wrong@firma.de')
    await user.type(screen.getByLabelText('auth.password'), 'wrong')
    await user.click(screen.getByRole('button', { name: 'auth.login' }))

    await waitFor(() => {
      expect(screen.getByText(key)).toBeInTheDocument()
    })
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('keeps a non-key error message readable instead of printing a raw key', async () => {
    const loginMock = vi.fn().mockRejectedValue(new Error('Something specific went wrong'))
    useAuthStore.setState({ login: loginMock })

    const user = userEvent.setup()
    renderLogin()

    await user.type(screen.getByLabelText('auth.email'), 'wrong@firma.de')
    await user.type(screen.getByLabelText('auth.password'), 'wrong')
    await user.click(screen.getByRole('button', { name: 'auth.login' }))

    await waitFor(() => {
      expect(screen.getByText('Something specific went wrong')).toBeInTheDocument()
    })
  })

  it('shows 2FA prompt when backend requires TOTP', async () => {
    // Mock login to throw 2FA_REQUIRED and set pendingToken
    const loginMock = vi.fn().mockImplementation(async () => {
      useAuthStore.setState({ pendingToken: 'pending-2fa-token' })
      throw new Error('2FA_REQUIRED')
    })
    useAuthStore.setState({ login: loginMock })

    const user = userEvent.setup()
    renderLogin()

    await user.type(screen.getByLabelText('auth.email'), '2fa@firma.de')
    await user.type(screen.getByLabelText('auth.password'), 'correct')
    await user.click(screen.getByRole('button', { name: 'auth.login' }))

    // Should switch to 2FA stage
    await waitFor(() => {
      expect(screen.getByLabelText('security.2fa.enterCode')).toBeInTheDocument()
    })
  })

  it('completes 2FA login with valid TOTP code', async () => {
    // Mock login to throw 2FA_REQUIRED
    const loginMock = vi.fn().mockImplementation(async () => {
      useAuthStore.setState({ pendingToken: 'pending-2fa-token' })
      throw new Error('2FA_REQUIRED')
    })
    const complete2FAMock = vi.fn().mockResolvedValue(undefined)
    useAuthStore.setState({ login: loginMock, complete2FALogin: complete2FAMock })

    const user = userEvent.setup()
    renderLogin()

    // First: trigger 2FA flow
    await user.type(screen.getByLabelText('auth.email'), '2fa@firma.de')
    await user.type(screen.getByLabelText('auth.password'), 'correct')
    await user.click(screen.getByRole('button', { name: 'auth.login' }))

    await waitFor(() => {
      expect(screen.getByLabelText('security.2fa.enterCode')).toBeInTheDocument()
    })

    // Then: enter TOTP code
    await user.type(screen.getByLabelText('security.2fa.enterCode'), '123456')
    await user.click(screen.getByRole('button', { name: 'security.2fa.verify' }))

    await waitFor(() => {
      expect(complete2FAMock).toHaveBeenCalledWith('pending-2fa-token', '123456', true)
    })
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true })
    })
  })
})
