# LuxSuv Booking Application

A full-stack luxury SUV booking platform featuring a passwordless guest booking system with email verification and traditional user accounts.

## 🏗️ Architecture Overview

### Tech Stack

**Backend**
- **API**: Go with Chi router
- **Database**: PostgreSQL with connection pooling
- **Authentication**: JWT-based with passwordless guest access
- **Email**: SMTP (development) / MailerSend (production)
- **Rate Limiting**: PostgreSQL-backed with configurable windows

**Frontend**
- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite for fast development and optimized builds
- **Routing**: TanStack Router for type-safe routing
- **State Management**: TanStack Query for server state management
- **Styling**: Tailwind CSS with responsive design
- **Form Handling**: React Hook Form with Zod validation
- **UI Components**: Headless UI with custom design system

### Key Features

- **Passwordless Guest Booking**: Create bookings without account registration
- **Email Verification**: 6-digit codes and magic links for guest access
- **Session Management**: JWT-based sessions with automatic renewal
- **Rate Limiting**: Configurable rate limits on sensitive endpoints
- **Idempotency**: Duplicate request protection with custom keys
- **Real-time Updates**: Optimistic updates with server synchronization
- **Responsive Design**: Mobile-first approach with desktop optimization
- **Type Safety**: Full TypeScript coverage from API to UI components

## 🚀 Quick Start

### Prerequisites

- **Go** 1.21+
- **Node.js** 18+
- **PostgreSQL** 15+
- **SMTP Server** (Mailpit for development)

### Backend Setup

1. **Clone and setup database**
   ```bash
   git clone <repository>
   cd luxsuv-bookings
   
   # Start PostgreSQL
   make db/up
   
   # Run migrations
   make migrate/up
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start the API server**
   ```bash
   make run
   # Server starts on http://localhost:8080
   ```

### Frontend Setup

1. **Install dependencies**
   ```bash
   cd frontend
   npm install
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env.local
   # Edit with your API URL
   ```

3. **Start development server**
   ```bash
   npm run dev
   # Frontend starts on http://localhost:5173
   ```

### Development Tools

**Mailpit** (Email testing in development)
```bash
# Install Mailpit
go install github.com/axllent/mailpit@latest

# Start mail server
mailpit --listen 0.0.0.0:8025 --smtp 0.0.0.0:1025

# View emails at http://localhost:8025
```

## 📊 Guest Booking Flow

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant API
    participant Email

    User->>Frontend: Create booking
    Frontend->>API: POST /v1/guest/bookings
    API-->>Frontend: {id, manage_token}
    
    User->>Frontend: Request email access
    Frontend->>API: POST /v1/guest/access/request
    API->>Email: Send verification code + magic link
    
    User->>Frontend: Enter verification code
    Frontend->>API: POST /v1/guest/access/verify
    API-->>Frontend: {session_token}
    
    Frontend->>API: Authenticated requests
    Note over Frontend: Session stored in localStorage
```

## 🔧 Environment Configuration

### Backend (.env)

```env
# Server
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/luxsuv-co?sslmode=disable

# JWT
JWT_SECRET=your-secure-secret-key

# Email (Development - Mailpit)
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_FROM=noreply@luxsuv.local
SMTP_USER=
SMTP_PASS=
SMTP_USE_TLS=0

# Email (Production - MailerSend)
MAILERSEND_API_KEY=your-mailersend-api-key
MAILER_FROM=noreply@yourdomain.com
```

### Frontend (.env.local)

```env
VITE_API_URL=http://localhost:8080
VITE_APP_NAME=LuxSuv Bookings
VITE_ENVIRONMENT=development

# Optional: Analytics, monitoring
VITE_GA_MEASUREMENT_ID=
VITE_SENTRY_DSN=
```

## 🎯 API Endpoints

### Guest Access (Passwordless Authentication)

#### Request Access Code
```http
POST /v1/guest/access/request
Content-Type: application/json

{
  "email": "user@example.com"
}
```

**Response**: `200 OK`
```json
{
  "message": "Access code sent to your email"
}
```

#### Verify Access Code
```http
POST /v1/guest/access/verify
Content-Type: application/json

{
  "email": "user@example.com",
  "code": "123456"
}
```

**Response**: `200 OK`
```json
{
  "session_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 1800
}
```

#### Magic Link Access
```http
POST /v1/guest/access/magic?token=550e8400-e29b-41d4-a716-446655440000
```

### Guest Bookings

#### Create Booking
```http
POST /v1/guest/bookings
Content-Type: application/json
Idempotency-Key: unique-key-123 (optional)

{
  "rider_name": "John Doe",
  "rider_email": "john@example.com",
  "rider_phone": "+1234567890",
  "pickup": "SFO Terminal 1", 
  "dropoff": "Downtown Hotel",
  "scheduled_at": "2025-12-01T15:30:00Z",
  "notes": "2 large bags",
  "passengers": 2,
  "luggages": 2,
  "ride_type": "per_ride"
}
```

**Response**: `201 Created`
```json
{
  "id": 123,
  "manage_token": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "scheduled_at": "2025-12-01T15:30:00Z"
}
```

#### List My Bookings (Session Required)
```http
GET /v1/guest/bookings?limit=20&offset=0&status=pending
Authorization: Bearer <session_token>
```

#### Get Single Booking
```http
# With session
GET /v1/guest/bookings/123
Authorization: Bearer <session_token>

# With manage token
GET /v1/guest/bookings/123?manage_token=550e8400-e29b-41d4-a716-446655440000
```

#### Update Booking
```http
PATCH /v1/guest/bookings/123?manage_token=<token>
Content-Type: application/json

{
  "notes": "Updated notes",
  "passengers": 3
}
```

#### Cancel Booking
```http
DELETE /v1/guest/bookings/123?manage_token=<token>
```

## ⚛️ Frontend Architecture

### Project Structure

```
frontend/
├── src/
│   ├── components/          # Reusable UI components
│   │   ├── ui/             # Base UI components
│   │   ├── forms/          # Form components
│   │   └── layouts/        # Layout components
│   ├── hooks/              # Custom React hooks
│   ├── lib/                # Utilities and configurations
│   │   ├── api.ts          # API client setup
│   │   ├── auth.ts         # Authentication utilities
│   │   └── utils.ts        # General utilities
│   ├── routes/             # Route components
│   │   ├── guest/          # Guest booking routes
│   │   ├── auth/           # Authentication routes
│   │   └── dashboard/      # User dashboard routes
│   ├── stores/             # Zustand stores for client state
│   ├── styles/             # Tailwind and global styles
│   └── types/              # TypeScript type definitions
├── public/                 # Static assets
└── package.json
```

### State Management Strategy

**Server State**: TanStack Query
- API data caching and synchronization
- Optimistic updates for bookings
- Background refetching
- Error boundary integration

**Client State**: Zustand
- Authentication state
- UI preferences
- Form state (complex forms)
- Navigation state

**Form State**: React Hook Form + Zod
- Type-safe form validation
- Real-time validation feedback
- Integration with API error responses

### Key Frontend Components

#### API Client (`src/lib/api.ts`)

```typescript
import { QueryClient } from '@tanstack/react-query'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        if (error?.status === 401) return false
        return failureCount < 3
      },
      staleTime: 5 * 60 * 1000, // 5 minutes
    },
  },
})

class BookingAPI {
  private baseURL = import.meta.env.VITE_API_URL

  async request(endpoint: string, options: RequestInit = {}) {
    const token = getAuthToken()
    
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      },
    })

    if (!response.ok) {
      throw new APIError(response.status, await response.json())
    }

    return response.json()
  }

  // Booking methods...
  createBooking = (data: CreateBookingRequest) =>
    this.request('/v1/guest/bookings', {
      method: 'POST',
      body: JSON.stringify(data),
    })
}
```

#### Authentication Hook (`src/hooks/useAuth.ts`)

```typescript
import { useMutation, useQuery } from '@tanstack/react-query'
import { useAuthStore } from '../stores/auth'

export function useAuth() {
  const { user, setUser, clearAuth } = useAuthStore()

  const requestAccessMutation = useMutation({
    mutationFn: (email: string) =>
      api.requestGuestAccess(email),
    onSuccess: () => {
      toast.success('Access code sent to your email!')
    },
  })

  const verifyAccessMutation = useMutation({
    mutationFn: ({ email, code }: { email: string; code: string }) =>
      api.verifyGuestAccess(email, code),
    onSuccess: (data) => {
      setUser({
        email: data.email,
        sessionToken: data.session_token,
        expiresAt: Date.now() + data.expires_in * 1000,
      })
      router.navigate('/guest/bookings')
    },
  })

  return {
    user,
    isAuthenticated: !!user?.sessionToken,
    requestAccess: requestAccessMutation.mutate,
    verifyAccess: verifyAccessMutation.mutate,
    logout: clearAuth,
  }
}
```

#### Booking Queries (`src/hooks/useBookings.ts`)

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

export function useBookings() {
  const queryClient = useQueryClient()

  const bookingsQuery = useQuery({
    queryKey: ['bookings'],
    queryFn: () => api.listBookings(),
    enabled: isAuthenticated,
  })

  const createBookingMutation = useMutation({
    mutationFn: api.createBooking,
    onSuccess: (newBooking) => {
      queryClient.setQueryData(['bookings'], (old: Booking[] = []) => 
        [newBooking, ...old]
      )
      toast.success('Booking created successfully!')
    },
    onError: (error: APIError) => {
      toast.error(error.message)
    },
  })

  return {
    bookings: bookingsQuery.data ?? [],
    isLoading: bookingsQuery.isLoading,
    createBooking: createBookingMutation.mutate,
    isCreating: createBookingMutation.isPending,
  }
}
```

### Routing Configuration

#### Route Tree (`src/routes/__root.tsx`)

```typescript
import { createRootRoute } from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'

const queryClient = new QueryClient()

export const Route = createRootRoute({
  component: RootComponent,
})

function RootComponent() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="min-h-screen bg-gray-50">
        <Outlet />
        <Toaster position="top-right" />
      </div>
    </QueryClientProvider>
  )
}
```

#### Guest Routes (`src/routes/guest/bookings/index.tsx`)

```typescript
import { createFileRoute } from '@tanstack/react-router'
import { BookingList } from '../../../components/BookingList'
import { useAuth } from '../../../hooks/useAuth'

export const Route = createFileRoute('/guest/bookings/')({
  component: GuestBookingsPage,
  beforeLoad: ({ context }) => {
    if (!context.auth.isAuthenticated) {
      throw redirect('/guest/access')
    }
  },
})

function GuestBookingsPage() {
  const { bookings, isLoading } = useBookings()

  if (isLoading) {
    return <BookingListSkeleton />
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">My Bookings</h1>
      <BookingList bookings={bookings} />
    </div>
  )
}
```

## 🎨 UI Design System

### Tailwind Configuration

```typescript
// tailwind.config.js
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
        },
        gray: {
          50: '#f9fafb',
          100: '#f3f4f6',
          900: '#111827',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
  ],
}
```

### Component Library Structure

```typescript
// src/components/ui/Button.tsx
interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  isLoading?: boolean
}

export function Button({ 
  variant = 'primary', 
  size = 'md', 
  isLoading,
  children,
  ...props 
}: ButtonProps) {
  return (
    <button
      className={cn(
        'inline-flex items-center justify-center rounded-lg font-medium transition-colors',
        'focus:outline-none focus:ring-2 focus:ring-offset-2',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        {
          'bg-primary-600 text-white hover:bg-primary-700 focus:ring-primary-500': variant === 'primary',
          'bg-gray-200 text-gray-900 hover:bg-gray-300 focus:ring-gray-500': variant === 'secondary',
          'text-gray-700 hover:bg-gray-100 focus:ring-gray-500': variant === 'ghost',
        },
        {
          'px-3 py-1.5 text-sm': size === 'sm',
          'px-4 py-2 text-base': size === 'md',
          'px-6 py-3 text-lg': size === 'lg',
        }
      )}
      disabled={isLoading}
      {...props}
    >
      {isLoading && <Spinner className="mr-2 h-4 w-4" />}
      {children}
    </button>
  )
}
```

## 🔒 Error Handling & Validation

### API Error Types

```typescript
// src/types/api.ts
export interface APIError extends Error {
  status: number
  code?: string
  details?: string
}

export const ERROR_CODES = {
  INVALID_INPUT: 'INVALID_INPUT',
  UNAUTHORIZED: 'UNAUTHORIZED',
  RATE_LIMIT_EXCEEDED: 'RATE_LIMIT_EXCEEDED',
  PAST_DATETIME: 'PAST_DATETIME',
} as const
```

### Form Validation Schemas

```typescript
// src/lib/validations.ts
import { z } from 'zod'

export const createBookingSchema = z.object({
  rider_name: z.string().min(1, 'Name is required').max(100),
  rider_email: z.string().email('Invalid email format'),
  rider_phone: z.string().regex(/^\+?[\d\s-()]+$/, 'Invalid phone format'),
  pickup: z.string().min(1, 'Pickup location is required'),
  dropoff: z.string().min(1, 'Dropoff location is required'),
  scheduled_at: z.date().refine(
    (date) => date > new Date(),
    'Scheduled time must be in the future'
  ),
  passengers: z.number().min(1).max(8),
  luggages: z.number().min(0).max(10),
  ride_type: z.enum(['per_ride', 'hourly']),
  notes: z.string().optional(),
})

export type CreateBookingInput = z.infer<typeof createBookingSchema>
```

### Error Boundary Component

```typescript
// src/components/ErrorBoundary.tsx
import { QueryErrorResetBoundary } from '@tanstack/react-query'
import { ErrorBoundary } from 'react-error-boundary'

export function AppErrorBoundary({ children }: { children: React.ReactNode }) {
  return (
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <ErrorBoundary
          onReset={reset}
          fallbackRender={({ error, resetErrorBoundary }) => (
            <div className="min-h-screen flex items-center justify-center">
              <div className="text-center">
                <h2 className="text-2xl font-bold mb-4">Something went wrong</h2>
                <p className="text-gray-600 mb-6">{error.message}</p>
                <Button onClick={resetErrorBoundary}>Try again</Button>
              </div>
            </div>
          )}
        >
          {children}
        </ErrorBoundary>
      )}
    </QueryErrorResetBoundary>
  )
}
```

## 🧪 Testing Strategy

### Backend Testing

```bash
# Unit tests
go test ./internal/...

# Integration tests with test database
TEST_DATABASE_URL=postgres://test:test@localhost:5432/luxsuv_test go test ./...

# API testing with HTTP files
# Use test.http for manual API testing
```

### Frontend Testing

```bash
# Unit and component tests
npm run test

# E2E tests
npm run test:e2e

# Type checking
npm run type-check
```

#### Test Examples

```typescript
// src/hooks/__tests__/useBookings.test.ts
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useBookings } from '../useBookings'

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  )
}

describe('useBookings', () => {
  it('fetches bookings successfully', async () => {
    const { result } = renderHook(() => useBookings(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.bookings).toHaveLength(2)
    })
  })
})
```

## 🚀 Deployment

### Backend Deployment

**Docker Configuration**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o api ./cmd/api

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /app/api .
EXPOSE 8080
CMD ["./api"]
```

**Environment Variables (Production)**
```env
PORT=8080
DATABASE_URL=postgres://user:pass@host:5432/luxsuv?sslmode=require
JWT_SECRET=secure-production-secret
MAILERSEND_API_KEY=your-production-api-key
MAILER_FROM=noreply@yourdomain.com
```

### Frontend Deployment

**Build Configuration**
```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { TanStackRouterVite } from '@tanstack/router-vite-plugin'

export default defineConfig({
  plugins: [react(), TanStackRouterVite()],
  build: {
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          router: ['@tanstack/react-router'],
          query: ['@tanstack/react-query'],
        },
      },
    },
  },
})
```

**Static Hosting (Vercel/Netlify)**
```json
{
  "rewrites": [
    { "source": "/(.*)", "destination": "/index.html" }
  ]
}
```

## 📊 Performance Considerations

### Backend Optimizations

- **Connection Pooling**: PostgreSQL connection pool with configurable limits
- **Query Optimization**: Indexed columns for frequent lookups
- **Rate Limiting**: PostgreSQL-backed rate limiting to prevent abuse
- **Caching**: Consider Redis for session storage in production

### Frontend Optimizations

- **Code Splitting**: Route-based and component-based code splitting
- **Bundle Analysis**: Regular bundle size monitoring
- **Image Optimization**: WebP format with fallbacks
- **Service Worker**: Offline functionality for booking management

### Monitoring & Analytics

```typescript
// src/lib/analytics.ts
import { analytics } from './firebase'

export function trackBookingCreated(bookingId: string) {
  analytics.track('booking_created', {
    booking_id: bookingId,
    timestamp: new Date().toISOString(),
  })
}

export function trackEmailVerification(email: string) {
  analytics.track('email_verification_requested', {
    email_domain: email.split('@')[1],
  })
}
```

## 🔧 Development Workflow

### Git Workflow

```bash
# Feature development
git checkout -b feature/booking-notifications
git commit -m "feat: add email notifications for booking updates"

# Backend changes
git commit -m "backend: add notification service"

# Frontend changes  
git commit -m "frontend: add notification toast component"
```

### Code Quality

**Pre-commit Hooks**
```json
{
  "husky": {
    "hooks": {
      "pre-commit": "lint-staged"
    }
  },
  "lint-staged": {
    "*.{ts,tsx}": ["eslint --fix", "prettier --write"],
    "*.go": ["gofmt -w", "golint"]
  }
}
```

### Development Commands

```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "test": "vitest",
    "test:e2e": "playwright test",
    "lint": "eslint src --ext ts,tsx",
    "lint:fix": "eslint src --ext ts,tsx --fix",
    "type-check": "tsc --noEmit"
  }
}
```

## 📈 Future Enhancements

### Planned Features

1. **Real-time Updates**: WebSocket integration for live booking status
2. **Mobile App**: React Native app with shared API client
3. **Payment Integration**: Stripe integration for booking payments
4. **Driver Dashboard**: Separate interface for driver management
5. **Analytics Dashboard**: Booking metrics and business intelligence
6. **Multi-language**: i18n support for international markets

### Technical Improvements

1. **GraphQL API**: Consider GraphQL for more efficient data fetching
2. **Microservices**: Split into booking, auth, and notification services
3. **Event Sourcing**: Implement event sourcing for audit trails
4. **Caching Layer**: Redis integration for improved performance
5. **CDN Integration**: CloudFront for global asset delivery

## 🆘 Troubleshooting

### Common Issues

**Backend**
- **Database Connection**: Check PostgreSQL is running and credentials are correct
- **Email Not Sending**: Verify SMTP configuration or MailerSend API key
- **CORS Issues**: Ensure frontend origin is listed in CORS configuration

**Frontend**
- **Build Errors**: Clear node_modules and reinstall dependencies
- **API Connection**: Check VITE_API_URL environment variable
- **Route Issues**: Ensure TanStack Router file-based routing structure is correct

### Debug Commands

```bash
# Backend
make db/psql  # Connect to database
go run ./cmd/api -debug  # Start with debug logging

# Frontend
npm run build -- --mode development  # Debug build
npm run dev -- --debug  # Start with debug mode
```

## 🤝 Contributing

### Development Setup

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

### Code Style

- **Backend**: Follow Go best practices, use `gofmt` and `golint`
- **Frontend**: Use ESLint and Prettier configurations
- **Commits**: Follow conventional commit format

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature  
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
```

---

## 📞 Support

For questions or support:
- **Technical Issues**: Create an issue in the repository
- **Business Inquiries**: Contact the development team
- **Security Issues**: Report privately to security@luxsuv.com

Built with ❤️ by the LuxSuv development team.