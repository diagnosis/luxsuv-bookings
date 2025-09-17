LuxSuv Booking Application
A full-stack luxury SUV booking platform featuring a passwordless guest booking system with email verification and traditional user accounts.

🏗️ Architecture Overview
Tech Stack
Backend

API: Go with Chi router
Database: PostgreSQL with connection pooling
Authentication: JWT-based with passwordless guest access
Email: SMTP (development) / MailerSend (production)
Rate Limiting: PostgreSQL-backed with configurable windows
Frontend

Framework: React 18 with TypeScript
Build Tool: Vite for fast development and optimized builds
Routing: TanStack Router for type-safe routing
State Management: TanStack Query for server state management
Styling: Tailwind CSS with responsive design
Form Handling: React Hook Form with Zod validation
UI Components: Headless UI with custom design system
Key Features
Passwordless Guest Booking: Create bookings without account registration
Email Verification: 6-digit codes and magic links for guest access
Session Management: JWT-based sessions with automatic renewal
Rate Limiting: Configurable rate limits on sensitive endpoints
Idempotency: Duplicate request protection with custom keys
Real-time Updates: Optimistic updates with server synchronization
Responsive Design: Mobile-first approach with desktop optimization
Type Safety: Full TypeScript coverage from API to UI components
🚀 Quick Start
Prerequisites
Go 1.21+
Node.js 18+
PostgreSQL 15+
SMTP Server (Mailpit for development)
Backend Setup
Clone and setup database

git clone <repository>
cd luxsuv-bookings

# Start PostgreSQL
make db/up

# Run migrations
make migrate/up
Configure environment

cp .env.example .env
# Edit .env with your configuration
Start the API server

make run
# Server starts on http://localhost:8080
Frontend Setup
Install dependencies

cd frontend
npm install
Configure environment

cp .env.example .env.local
# Edit with your API URL
Start development server

npm run dev
# Frontend starts on http://localhost:5173
Development Tools
Mailpit (Email testing in development)

# Install Mailpit
go install github.com/axllent/mailpit@latest

# Start mail server
mailpit --listen 0.0.0.0:8025 --smtp 0.0.0.0:1025

# View emails at http://localhost:8025
📊 Guest Booking Flow
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
🔧 Environment Configuration
Backend (.env)
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
Frontend (.env.local)
VITE_API_URL=http://localhost:8080
VITE_APP_NAME=LuxSuv Bookings
VITE_ENVIRONMENT=development

# Optional: Analytics, monitoring
VITE_GA_MEASUREMENT_ID=
VITE_SENTRY_DSN=
🎯 API Endpoints
Guest Access (Passwordless Authentication)
Request Access Code
POST /v1/guest/access/request
Content-Type: application/json

{
  "email": "user@example.com"
}
Response: 200 OK

{
  "message": "Access code sent to your email"
}
Verify Access Code
POST /v1/guest/access/verify
Content-Type: application/json

{
  "email": "user@example.com",
  "code": "123456"
}
Response: 200 OK

{
  "session_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 1800
}
Magic Link Access
POST /v1/guest/access/magic?token=550e8400-e29b-41d4-a716-446655440000
Guest Bookings
Create Booking
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
Response: 201 Created

{
  "id": 123,
  "manage_token": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "scheduled_at": "2025-12-01T15:30:00Z"
}
List My Bookings (Session Required)
GET /v1/guest/bookings?limit=20&offset=0&status=pending
Authorization: Bearer <session_token>
Get Single Booking
# With session
GET /v1/guest/bookings/123
Authorization: Bearer <session_token>

# With manage token
GET /v1/guest/bookings/123?manage_token=550e8400-e29b-41d4-a716-446655440000
Update Booking
PATCH /v1/guest/bookings/123?manage_token=<token>
Content-Type: application/json

{
  "notes": "Updated notes",
  "passengers": 3
}
Cancel Booking
DELETE /v1/guest/bookings/123?manage_token=<token>
⚛️ Frontend Architecture
Project Structure
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
State Management Strategy
Server State: TanStack Query

API data caching and synchronization
Optimistic updates for bookings
Background refetching
Error boundary integration
Client State: Zustand

Authentication state
UI preferences
Form state (complex forms)
Navigation state
Form State: React Hook Form + Zod

Type-safe form validation
Real-time validation feedback
Integration with API error responses
Key Frontend Components
API Client (src/lib/api.ts)
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
Authentication Hook (src/hooks/useAuth.ts)
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
Booking Queries (src/hooks/useBookings.ts)
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
Routing Configuration
Route Tree (src/routes/__root.tsx)
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
Guest Routes (src/routes/guest/bookings/index.tsx)
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
🎨 UI Design System
Tailwind Configuration
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
Component Library Structure
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
🔒 Error Handling & Validation
API Error Types
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
Form Validation Schemas
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
Error Boundary Component
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
🧪 Testing Strategy
Backend Testing
# Unit tests
go test ./internal/...

# Integration tests with test database
TEST_DATABASE_URL=postgres://test:test@localhost:5432/luxsuv_test go test ./...

# API testing with HTTP files
# Use test.http for manual API testing
Frontend Testing
# Unit and component tests
npm run test

# E2E tests
npm run test:e2e

# Type checking
npm run type-check
Test Examples
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
🚀 Deployment
Backend Deployment
Docker Configuration

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
Environment Variables (Production)

PORT=8080
DATABASE_URL=postgres://user:pass@host:5432/luxsuv?sslmode=require
JWT_SECRET=secure-production-secret
MAILERSEND_API_KEY=your-production-api-key
MAILER_FROM=noreply@yourdomain.com
Frontend Deployment
Build Configuration

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
Static Hosting (Vercel/Netlify)

{
  "rewrites": [
    { "source": "/(.*)", "destination": "/index.html" }
  ]
}
📊 Performance Considerations
Backend Optimizations
Connection Pooling: PostgreSQL connection pool with configurable limits
Query Optimization: Indexed columns for frequent lookups
Rate Limiting: PostgreSQL-backed rate limiting to prevent abuse
Caching: Consider Redis for session storage in production
Frontend Optimizations
Code Splitting: Route-based and component-based code splitting
Bundle Analysis: Regular bundle size monitoring
Image Optimization: WebP format with fallbacks
Service Worker: Offline functionality for booking management
Monitoring & Analytics
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
🔧 Development Workflow
Git Workflow
# Feature development
git checkout -b feature/booking-notifications
git commit -m "feat: add email notifications for booking updates"

# Backend changes
git commit -m "backend: add notification service"

# Frontend changes  
git commit -m "frontend: add notification toast component"
Code Quality
Pre-commit Hooks

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
Development Commands
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
📈 Future Enhancements
Planned Features
Real-time Updates: WebSocket integration for live booking status
Mobile App: React Native app with shared API client
Payment Integration: Stripe integration for booking payments
Driver Dashboard: Separate interface for driver management
Analytics Dashboard: Booking metrics and business intelligence
Multi-language: i18n support for international markets
Technical Improvements
GraphQL API: Consider GraphQL for more efficient data fetching
Microservices: Split into booking, auth, and notification services
Event Sourcing: Implement event sourcing for audit trails
Caching Layer: Redis integration for improved performance
CDN Integration: CloudFront for global asset delivery
🆘 Troubleshooting
Common Issues
Backend

Database Connection: Check PostgreSQL is running and credentials are correct
Email Not Sending: Verify SMTP configuration or MailerSend API key
CORS Issues: Ensure frontend origin is listed in CORS configuration
Frontend

Build Errors: Clear node_modules and reinstall dependencies
API Connection: Check VITE_API_URL environment variable
Route Issues: Ensure TanStack Router file-based routing structure is correct
Debug Commands
# Backend
make db/psql  # Connect to database
go run ./cmd/api -debug  # Start with debug logging

# Frontend
npm run build -- --mode development  # Debug build
npm run dev -- --debug  # Start with debug mode
🤝 Contributing
Development Setup
Fork the repository
Create a feature branch
Make your changes
Add tests for new functionality
Ensure all tests pass
Submit a pull request
Code Style
Backend: Follow Go best practices, use gofmt and golint
Frontend: Use ESLint and Prettier configurations
Commits: Follow conventional commit format
Pull Request Template
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
📞 Support
For questions or support:

Technical Issues: Create an issue in the repository
Business Inquiries: Contact the development team
Security Issues: Report privately to security@luxsuv.com
Built with ❤️ by the LuxSuv development team.

---

# 🌟 Multi-App Frontend Architecture

This section provides a comprehensive guide for implementing multiple React frontends for the LuxSuv booking platform, each optimized for specific user roles and use cases.

## 🏢 Frontend Applications Overview

The LuxSuv platform consists of three separate React applications, each tailored for different user types:

### 🚗 Rider App (Customer-Facing)
**Purpose**: Customer booking interface for riders
**Users**: Guests, Registered Riders
**Deployment**: Web app (mobile-responsive)
**Future**: Mobile app (React Native)

**Key Features:**
- **Passwordless Guest Booking**: No-friction booking experience
- **User Registration/Login**: Account management with booking history
- **Real-time Booking Updates**: Live status updates and notifications
- **Mobile-Optimized**: Touch-friendly interface, mobile-first design
- **Booking Management**: View, edit, cancel bookings
- **Payment Integration**: Stripe payment processing

### 🏢 Admin/Dispatcher Portal (Operations)
**Purpose**: Operations management for admin staff and dispatchers
**Users**: Admins, Dispatchers
**Deployment**: Web app (desktop-optimized)

**Key Features:**
- **Booking Management**: View all bookings, advanced filtering
- **Driver Assignment**: Manual and automatic assignment tools
- **Real-time Dashboard**: Live operations overview
- **User Management**: Manage riders, drivers, and staff
- **Analytics & Reporting**: Business intelligence and insights
- **System Configuration**: Platform settings and configuration

### 📱 Driver App (Driver-Facing)
**Purpose**: Driver interface for managing assignments and trips
**Users**: Drivers
**Deployment**: PWA (progressive web app)
**Future**: Native mobile app (React Native)

**Key Features:**
- **Assignment Management**: Accept/decline ride assignments
- **Trip Navigation**: GPS integration and route optimization
- **Availability Toggle**: Online/offline status management
- **Earnings Tracking**: Trip history and payment information
- **Real-time Communication**: Chat with dispatch and riders
- **Mobile-First**: Optimized for mobile use in vehicles

## 🏗️ Monorepo Architecture

### Project Structure
```
frontend/
├── apps/                          # Individual applications
│   ├── rider/                     # Rider customer app
│   │   ├── src/
│   │   ├── public/
│   │   ├── package.json
│   │   ├── vite.config.ts
│   │   └── tailwind.config.js
│   ├── admin/                     # Admin/dispatcher portal
│   │   ├── src/
│   │   ├── public/
│   │   ├── package.json
│   │   ├── vite.config.ts
│   │   └── tailwind.config.js
│   └── driver/                    # Driver mobile app
│       ├── src/
│       ├── public/
│       ├── package.json
│       ├── vite.config.ts
│       └── tailwind.config.js
├── packages/                      # Shared packages
│   ├── ui/                        # Shared UI components
│   │   ├── src/
│   │   │   ├── components/        # Base design system
│   │   │   ├── hooks/            # UI hooks
│   │   │   └── styles/           # Shared styles
│   │   ├── package.json
│   │   └── tsconfig.json
│   ├── api/                       # Shared API client
│   │   ├── src/
│   │   │   ├── client.ts         # Base API client
│   │   │   ├── auth.ts           # Auth endpoints
│   │   │   ├── bookings.ts       # Booking endpoints
│   │   │   ├── admin.ts          # Admin endpoints
│   │   │   ├── driver.ts         # Driver endpoints
│   │   │   └── types/            # Shared types
│   │   ├── package.json
│   │   └── tsconfig.json
│   ├── utils/                     # Shared utilities
│   │   ├── src/
│   │   │   ├── validations/      # Zod schemas
│   │   │   ├── formatting/       # Date, currency, etc.
│   │   │   ├── constants/        # App constants
│   │   │   └── helpers/          # Utility functions
│   │   ├── package.json
│   │   └── tsconfig.json
│   └── config/                    # Shared configuration
│       ├── eslint-config/        # ESLint configs
│       ├── tailwind-config/      # Tailwind configs
│       └── typescript-config/    # TypeScript configs
├── package.json                   # Root package.json (workspace)
├── pnpm-workspace.yaml           # pnpm workspace config
├── turbo.json                    # Turborepo config
└── README.md                     # This guide
```

### Shared Tech Stack
```json
{
  "framework": "React 19 with TypeScript",
  "build_tool": "Vite for fast development",
  "routing": "TanStack Router (file-based)",
  "state_management": {
    "server_state": "TanStack Query (React Query)",
    "client_state": "Zustand stores",
    "form_state": "React Hook Form + Zod"
  },
  "monorepo": "pnpm workspaces + Turborepo",
  "styling": {
    "framework": "Tailwind CSS",
    "components": "Headless UI + Custom Design System",
    "icons": "Lucide React"
  },
  "validation": "Zod schemas",
  "testing": {
    "unit": "Vitest + Testing Library",
    "e2e": "Playwright",
    "type_checking": "TypeScript"
  }
}
```

## 🚀 Quick Start (Monorepo Setup)

### 1. Initialize Monorepo
```bash
# Create frontend directory
mkdir frontend && cd frontend

# Initialize root package.json
npm init -y

# Install pnpm globally (if not installed)
npm install -g pnpm

# Create workspace configuration
echo 'packages:
  - "apps/*"
  - "packages/*"' > pnpm-workspace.yaml
```

### 2. Install Global Dependencies
```bash
# Install Turborepo for monorepo management
pnpm add -D turbo

# Install shared dev dependencies
pnpm add -D typescript @types/node
pnpm add -D eslint prettier
pnpm add -D @typescript-eslint/eslint-plugin
pnpm add -D @typescript-eslint/parser
```

### 3. Create Apps
```bash
# Create rider app
mkdir -p apps/rider && cd apps/rider
pnpm create vite . --template react-ts
cd ../..

# Create admin portal
mkdir -p apps/admin && cd apps/admin  
pnpm create vite . --template react-ts
cd ../..

# Create driver app
mkdir -p apps/driver && cd apps/driver
pnpm create vite . --template react-ts
cd ../..
```

### 4. Setup Shared Packages
```bash
# Create shared UI package
mkdir -p packages/ui/src
cd packages/ui
pnpm init
# Add dependencies for UI package
pnpm add react react-dom @types/react @types/react-dom
pnpm add tailwindcss @headlessui/react lucide-react clsx
cd ../..

# Create shared API package
mkdir -p packages/api/src
cd packages/api
pnpm init
# Add dependencies for API package
pnpm add @tanstack/react-query zod
cd ../..

# Create shared utils package
mkdir -p packages/utils/src  
cd packages/utils
pnpm init
pnpm add date-fns zod
cd ../..
```

### 5. Configure Turborepo
```json
// turbo.json
{
  "pipeline": {
    "build": {
      "dependsOn": ["^build"],
      "outputs": ["dist/**"]
    },
    "dev": {
      "cache": false
    },
    "lint": {},
    "test": {
      "dependsOn": ["^build"]
    },
    "type-check": {
      "dependsOn": ["^build"]
    }
  }
}
```

### 6. Root Package.json Scripts
```json
{
  "name": "luxsuv-frontend",
  "private": true,
  "scripts": {
    "dev": "turbo run dev --parallel",
    "dev:rider": "turbo run dev --filter=rider",
    "dev:admin": "turbo run dev --filter=admin",
    "dev:driver": "turbo run dev --filter=driver",
    "build": "turbo run build",
    "test": "turbo run test",
    "lint": "turbo run lint",
    "type-check": "turbo run type-check",
    "clean": "turbo run clean && rm -rf node_modules"
  },
  "devDependencies": {
    "turbo": "^1.13.0"
  }
}
```

## 🚗 Rider App Implementation

### App-Specific Structure
```
apps/rider/
├── public/                     # Static assets
│   ├── icons/                 # Rider app icons
│   ├── images/               # Rider-specific images
│   └── manifest.json         # PWA manifest
├── src/
│   ├── components/            # Rider-specific components
│   │   ├── forms/            # Form-specific components
│   │   │   ├── BookingForm.tsx
│   │   │   ├── GuestAccessForm.tsx
│   │   │   └── ProfileForm.tsx
│   │   ├── layouts/          # Layout components
│   │   │   ├── RiderLayout.tsx
│   │   │   ├── GuestLayout.tsx
│   │   │   └── MobileLayout.tsx
│   │   └── features/         # Rider-specific features
│   │       ├── booking/      # Booking flow
│   │       │   ├── BookingCard.tsx
│   │       │   ├── BookingHistory.tsx
│   │       │   └── BookingStatus.tsx
│   │       ├── guest/        # Guest flow
│   │       │   ├── GuestAccess.tsx
│   │       │   └── GuestBookings.tsx
│   │       └── profile/      # User profile
│   │           ├── ProfileSettings.tsx
│   │           └── PaymentMethods.tsx
│   ├── hooks/                # Custom React hooks
│   │   ├── api/              # Rider API hooks
│   │   │   ├── useRiderAuth.ts
│   │   │   ├── useBookings.ts
│   │   │   └── useGuestAccess.ts
│   │   └── business/         # Rider business logic
│   │       ├── useBookingValidation.ts
│   │       ├── useGuestSession.ts
│   │       └── usePayment.ts
│   ├── routes/               # Rider routes
│   │   ├── __root.tsx        # Root layout and providers
│   │   ├── index.tsx         # Rider landing page
│   │   ├── book.tsx          # Quick booking flow
│   │   ├── guest/            # Guest booking routes
│   │   │   ├── access.tsx
│   │   │   └── bookings/
│   │   ├── auth/             # Authentication
│   │   │   ├── login.tsx
│   │   │   ├── register.tsx
│   │   │   └── verify.tsx
│   │   ├── dashboard/        # User dashboard
│   │   │   ├── index.tsx
│   │   │   ├── bookings/
│   │   │   └── profile.tsx
│   │   └── booking/          # Booking management
│   │       ├── $id.tsx       # Booking details
│   │       └── history.tsx   # Booking history
│   ├── stores/               # Zustand stores for client state
│   │   ├── auth.ts           # Rider authentication
│   │   ├── booking.ts        # Booking state
│   │   └── ui.ts             # UI preferences
│   ├── styles/               # Rider-specific styles
│   │   ├── globals.css
│   │   └── mobile.css        # Mobile-specific styles
│   └── main.tsx              # Rider app entry point
└── package.json
```

### Rider App Configuration
```json
// apps/rider/package.json
{
  "name": "luxsuv-rider",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite --port 3001",
    "build": "tsc && vite build",
    "preview": "vite preview --port 3001"
  },
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "@tanstack/react-router": "^1.58.0",
    "@tanstack/react-query": "^5.56.0",
    "zustand": "^4.5.0",
    "react-hook-form": "^7.53.0",
    "@hookform/resolvers": "^3.9.0",
    "zod": "^3.23.0",
    "tailwindcss": "^3.4.0",
    "@headlessui/react": "^2.1.0",
    "lucide-react": "^0.446.0",
    "sonner": "^1.5.0",
    "date-fns": "^4.1.0",
    "@luxsuv/ui": "workspace:*",
    "@luxsuv/api": "workspace:*",
    "@luxsuv/utils": "workspace:*"
  },
  "devDependencies": {
    "@types/react": "^18.3.0",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.0",
    "typescript": "^5.5.0",
    "vite": "^5.4.0",
    "vitest": "^2.1.0"
  }
}
```

## 🏢 Admin/Dispatcher Portal Implementation

### Admin App Structure
```
apps/admin/
├── src/
│   ├── components/
│   │   ├── dashboard/        # Dashboard components
│   │   │   ├── MetricsCard.tsx
│   │   │   ├── BookingChart.tsx
│   │   │   └── RealtimeMap.tsx
│   │   ├── tables/           # Data tables
│   │   │   ├── BookingsTable.tsx
│   │   │   ├── DriversTable.tsx
│   │   │   └── UsersTable.tsx
│   │   ├── forms/            # Admin forms
│   │   │   ├── AssignDriverForm.tsx
│   │   │   ├── UserManagementForm.tsx
│   │   │   └── SystemConfigForm.tsx
│   │   └── layouts/
│   │       ├── AdminLayout.tsx
│   │       ├── Sidebar.tsx
│   │       └── Header.tsx
│   ├── hooks/
│   │   ├── api/              # Admin API hooks
│   │   │   ├── useAdminAuth.ts
│   │   │   ├── useBookingManagement.ts
│   │   │   ├── useDriverManagement.ts
│   │   │   ├── useUserManagement.ts
│   │   │   └── useAnalytics.ts
│   │   └── business/         # Admin business logic
│   │       ├── useDispatch.ts
│   │       ├── useAssignment.ts
│   │       └── useReporting.ts
│   ├── routes/
│   │   ├── __root.tsx
│   │   ├── index.tsx         # Admin dashboard
│   │   ├── login.tsx         # Admin login
│   │   ├── bookings/         # Booking management
│   │   │   ├── index.tsx     # All bookings
│   │   │   ├── $id.tsx       # Booking details
│   │   │   └── assign.tsx    # Driver assignment
│   │   ├── drivers/          # Driver management
│   │   │   ├── index.tsx     # All drivers
│   │   │   ├── $id.tsx       # Driver details
│   │   │   └── availability.tsx
│   │   ├── users/            # User management
│   │   │   ├── index.tsx
│   │   │   └── $id.tsx
│   │   ├── dispatch/         # Dispatch tools
│   │   │   ├── live.tsx      # Live dispatch view
│   │   │   ├── pending.tsx   # Pending assignments
│   │   │   └── history.tsx   # Assignment history
│   │   └── analytics/        # Business intelligence
│   │       ├── overview.tsx
│   │       ├── performance.tsx
│   │       └── reports.tsx
│   ├── stores/
│   │   ├── auth.ts           # Admin authentication
│   │   ├── dispatch.ts       # Dispatch state
│   │   └── booking.ts        # Booking form state
│   └── main.tsx
└── package.json
```

### Admin Dashboard Component
```tsx
// apps/admin/src/routes/index.tsx
import { createFileRoute } from '@tanstack/react-router'
import { BarChart, Users, Car, Clock, TrendingUp, AlertCircle } from 'lucide-react'
import { MetricsCard } from '../components/dashboard/MetricsCard'
import { BookingChart } from '../components/dashboard/BookingChart'
import { RecentBookings } from '../components/dashboard/RecentBookings'
import { DriverStatus } from '../components/dashboard/DriverStatus'
import { useAdminDashboard } from '../hooks/api/useAdminDashboard'

export const Route = createFileRoute('/')({
  component: AdminDashboard,
  beforeLoad: ({ context }) => {
    // Check admin authentication
    if (!context.auth.isAuthenticated || context.auth.user?.role !== 'admin') {
      throw redirect({ to: '/login' })
    }
  },
})

function AdminDashboard() {
  const { metrics, bookingStats, driverStats, isLoading } = useAdminDashboard()

  if (isLoading) {
    return <DashboardSkeleton />
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Operations Dashboard</h1>
        <p className="text-gray-600">Real-time overview of LuxSuv operations</p>
      </div>

      {/* Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <MetricsCard
          title="Active Bookings"
          value={metrics.activeBookings}
          change={metrics.activeBookingsChange}
          icon={<Clock className="w-6 h-6" />}
          color="blue"
        />
        <MetricsCard
          title="Available Drivers"
          value={metrics.availableDrivers}
          change={metrics.availableDriversChange}
          icon={<Car className="w-6 h-6" />}
          color="green"
        />
        <MetricsCard
          title="Total Revenue"
          value={`$${metrics.totalRevenue.toLocaleString()}`}
          change={metrics.revenueChange}
          icon={<TrendingUp className="w-6 h-6" />}
          color="indigo"
        />
        <MetricsCard
          title="Issues"
          value={metrics.issues}
          change={metrics.issuesChange}
          icon={<AlertCircle className="w-6 h-6" />}
          color="red"
        />
      </div>

      {/* Charts and Lists */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="bg-white p-6 rounded-xl shadow-sm border">
          <h3 className="text-lg font-semibold mb-4">Booking Trends</h3>
          <BookingChart data={bookingStats} />
        </div>
        
        <div className="bg-white p-6 rounded-xl shadow-sm border">
          <h3 className="text-lg font-semibold mb-4">Driver Status</h3>
          <DriverStatus data={driverStats} />
        </div>
      </div>

      {/* Recent Activity */}
      <div className="bg-white rounded-xl shadow-sm border">
        <div className="p-6 border-b">
          <h3 className="text-lg font-semibold">Recent Bookings</h3>
        </div>
        <RecentBookings />
      </div>
    </div>
  )
}
```

### Dispatch Assignment Component
```tsx
// apps/admin/src/components/dispatch/AssignmentPanel.tsx
import React, { useState } from 'react'
import { MapPin, User, Clock, CheckCircle, X } from 'lucide-react'
import { Button } from '@luxsuv/ui'
import { useDispatch } from '../../hooks/api/useDispatch'
import type { Booking, Driver } from '@luxsuv/api/types'

interface AssignmentPanelProps {
  booking: Booking
  availableDrivers: Driver[]
  onAssign: (bookingId: number, driverId: number) => void
  onClose: () => void
}

export const AssignmentPanel: React.FC<AssignmentPanelProps> = ({
  booking,
  availableDrivers,
  onAssign,
  onClose,
}) => {
  const [selectedDriver, setSelectedDriver] = useState<number | null>(null)
  const { assignBooking, isAssigning } = useDispatch()

  const handleAssign = () => {
    if (selectedDriver) {
      assignBooking({
        bookingId: booking.id,
        driverId: selectedDriver,
      }, {
        onSuccess: () => {
          onAssign(booking.id, selectedDriver)
          onClose()
        },
      })
    }
  }

  return (
    <div className="bg-white rounded-xl shadow-lg border p-6 max-w-2xl">
      {/* Booking Info */}
      <div className="flex justify-between items-start mb-6">
        <div>
          <h3 className="text-lg font-semibold">Assign Driver</h3>
          <p className="text-sm text-gray-600">Booking #{booking.id}</p>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X className="w-4 h-4" />
        </Button>
      </div>

      {/* Trip Details */}
      <div className="bg-gray-50 rounded-lg p-4 mb-6">
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <MapPin className="w-4 h-4 text-green-600" />
            <span className="text-sm font-medium">Pickup:</span>
            <span className="text-sm text-gray-700">{booking.pickup}</span>
          </div>
          <div className="flex items-center gap-2">
            <MapPin className="w-4 h-4 text-red-600" />
            <span className="text-sm font-medium">Dropoff:</span>
            <span className="text-sm text-gray-700">{booking.dropoff}</span>
          </div>
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-blue-600" />
            <span className="text-sm font-medium">Scheduled:</span>
            <span className="text-sm text-gray-700">
              {format(new Date(booking.scheduled_at), 'MMM d, h:mm a')}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <User className="w-4 h-4 text-purple-600" />
            <span className="text-sm font-medium">Passenger:</span>
            <span className="text-sm text-gray-700">{booking.rider_name}</span>
          </div>
        </div>
      </div>

      {/* Available Drivers */}
      <div className="mb-6">
        <h4 className="text-md font-medium mb-3">Available Drivers ({availableDrivers.length})</h4>
        <div className="space-y-2 max-h-64 overflow-y-auto">
          {availableDrivers.map((driver) => (
            <div
              key={driver.id}
              className={clsx(
                'p-3 rounded-lg border cursor-pointer transition-colors',
                selectedDriver === driver.id
                  ? 'border-primary-500 bg-primary-50'
                  : 'border-gray-200 hover:border-gray-300'
              )}
              onClick={() => setSelectedDriver(driver.id)}
            >
              <div className="flex justify-between items-center">
                <div>
                  <p className="font-medium">{driver.name}</p>
                  <p className="text-sm text-gray-600">{driver.vehicle_info}</p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-medium">{driver.distance_km}km away</p>
                  <p className="text-xs text-gray-500">ETA: {driver.eta_minutes}min</p>
                </div>
                {selectedDriver === driver.id && (
                  <CheckCircle className="w-5 h-5 text-primary-600 ml-2" />
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Actions */}
      <div className="flex gap-3">
        <Button
          onClick={handleAssign}
          isLoading={isAssigning}
          disabled={!selectedDriver}
          className="flex-1"
        >
          Assign Driver
        </Button>
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
      </div>
    </div>
  )
}
```

## 📱 Driver App Implementation

### Driver App Structure (Mobile-First)
```
apps/driver/
├── src/
│   ├── components/
│   │   ├── mobile/           # Mobile-optimized components
│   │   │   ├── MobileNav.tsx
│   │   │   ├── SwipeCard.tsx
│   │   │   └── BottomSheet.tsx
│   │   ├── assignments/      # Assignment management
│   │   │   ├── AssignmentCard.tsx
│   │   │   ├── AcceptRejectButtons.tsx
│   │   │   └── TripTimer.tsx
│   │   ├── navigation/       # GPS and navigation
│   │   │   ├── MapView.tsx
│   │   │   ├── RouteDisplay.tsx
│   │   │   └── LocationPicker.tsx
│   │   └── status/           # Driver status
│   │       ├── AvailabilityToggle.tsx
│   │       ├── EarningsDisplay.tsx
│   │       └── TripHistory.tsx
│   ├── hooks/
│   │   ├── api/              # Driver API hooks
│   │   │   ├── useDriverAuth.ts
│   │   │   ├── useAssignments.ts
│   │   │   ├── useTrips.ts
│   │   │   └── useLocation.ts
│   │   ├── mobile/           # Mobile-specific hooks
│   │   │   ├── useGeolocation.ts
│   │   │   ├── useOrientation.ts
│   │   │   └── useVibration.ts
│   │   └── business/         # Driver business logic
│   │       ├── useDriverStatus.ts
│   │       ├── useTripManagement.ts
│   │       └── useEarnings.ts
│   ├── routes/
│   │   ├── __root.tsx
│   │   ├── index.tsx         # Driver home/status
│   │   ├── login.tsx         # Driver login
│   │   ├── assignments/      # Assignment management
│   │   │   ├── index.tsx     # Available assignments
│   │   │   └── $id.tsx       # Assignment details
│   │   ├── trips/            # Active trips
│   │   │   ├── active.tsx    # Current trip
│   │   │   ├── navigation.tsx# GPS navigation
│   │   │   └── complete.tsx  # Trip completion
│   │   ├── earnings/         # Earnings and payments
│   │   │   ├── index.tsx
│   │   │   └── history.tsx
│   │   └── profile/          # Driver profile
│   │       ├── index.tsx
│   │       ├── vehicle.tsx
│   │       └── documents.tsx
│   └── main.tsx
```

### Driver Assignment Component (Mobile-Optimized)
```tsx
// apps/driver/src/components/assignments/AssignmentCard.tsx
import React, { useState, useEffect } from 'react'
import { MapPin, Clock, Users, DollarSign, Navigation } from 'lucide-react'
import { format } from 'date-fns'
import { Button } from '@luxsuv/ui'
import { useAssignments } from '../../hooks/api/useAssignments'
import type { Assignment } from '@luxsuv/api/types'

interface AssignmentCardProps {
  assignment: Assignment
  onAccept: (id: number) => void
  onDecline: (id: number) => void
}

export const AssignmentCard: React.FC<AssignmentCardProps> = ({
  assignment,
  onAccept,
  onDecline,
}) => {
  const [timeLeft, setTimeLeft] = useState(0)
  const { acceptAssignment, declineAssignment, isProcessing } = useAssignments()

  // Countdown timer for assignment expiration
  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now()
      const expiresAt = new Date(assignment.expires_at).getTime()
      const remaining = Math.max(0, expiresAt - now)
      setTimeLeft(remaining)
    }, 1000)

    return () => clearInterval(interval)
  }, [assignment.expires_at])

  const formatTimeLeft = (ms: number) => {
    const minutes = Math.floor(ms / 60000)
    const seconds = Math.floor((ms % 60000) / 1000)
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
  }

  return (
    <div className="bg-white rounded-2xl shadow-lg border border-gray-200 p-6 mx-4">
      {/* Urgency Indicator */}
      <div className="flex justify-between items-center mb-4">
        <div className="flex items-center gap-2">
          <div className={clsx(
            'w-3 h-3 rounded-full',
            timeLeft > 120000 ? 'bg-green-500' :
            timeLeft > 60000 ? 'bg-yellow-500' : 'bg-red-500'
          )} />
          <span className="text-sm font-medium text-gray-600">
            Expires in {formatTimeLeft(timeLeft)}
          </span>
        </div>
        <div className="text-right">
          <p className="text-lg font-bold text-green-600">${assignment.estimated_fare}</p>
          <p className="text-xs text-gray-500">Estimated</p>
        </div>
      </div>

      {/* Trip Info */}
      <div className="space-y-3 mb-6">
        <div className="flex items-start gap-3">
          <div className="w-2 h-2 rounded-full bg-green-500 mt-2" />
          <div className="flex-1">
            <p className="text-sm font-medium text-gray-900">Pickup</p>
            <p className="text-sm text-gray-600">{assignment.pickup}</p>
            <p className="text-xs text-gray-500">
              {assignment.distance_to_pickup}km • {assignment.eta_to_pickup}min
            </p>
          </div>
        </div>
        
        <div className="flex items-start gap-3">
          <div className="w-2 h-2 rounded-full bg-red-500 mt-2" />
          <div className="flex-1">
            <p className="text-sm font-medium text-gray-900">Dropoff</p>
            <p className="text-sm text-gray-600">{assignment.dropoff}</p>
            <p className="text-xs text-gray-500">
              {assignment.trip_distance}km • {assignment.trip_duration}min
            </p>
          </div>
        </div>
      </div>

      {/* Passenger Info */}
      <div className="bg-gray-50 rounded-lg p-3 mb-6">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-2">
            <Users className="w-4 h-4 text-gray-600" />
            <span className="text-sm text-gray-700">
              {assignment.passengers} passenger{assignment.passengers !== 1 ? 's' : ''}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-gray-600" />
            <span className="text-sm text-gray-700">
              {format(new Date(assignment.scheduled_at), 'h:mm a')}
            </span>
          </div>
        </div>
        {assignment.notes && (
          <p className="text-sm text-gray-600 mt-2">{assignment.notes}</p>
        )}
      </div>

      {/* Action Buttons */}
      <div className="flex gap-3">
        <Button
          onClick={() => onDecline(assignment.id)}
          variant="secondary"
          className="flex-1"
          disabled={isProcessing}
        >
          Decline
        </Button>
        <Button
          onClick={handleAssign}
          isLoading={isProcessing}
          className="flex-1 bg-green-600 hover:bg-green-700"
        >
          Accept Ride
        </Button>
      </div>
    </div>
  )
}
```

## 📦 Shared Packages Implementation

### Shared UI Package
```json
// packages/ui/package.json
{
  "name": "@luxsuv/ui",
  "version": "1.0.0",
  "main": "./dist/index.js",
  "module": "./dist/index.mjs",
  "types": "./dist/index.d.ts",
  "exports": {
    ".": {
      "import": "./dist/index.mjs",
      "require": "./dist/index.js",
      "types": "./dist/index.d.ts"
    },
    "./styles": "./dist/styles.css"
  },
  "scripts": {
    "build": "tsup src/index.ts --format cjs,esm --dts",
    "dev": "tsup src/index.ts --format cjs,esm --dts --watch",
    "lint": "eslint src --ext ts,tsx",
    "type-check": "tsc --noEmit"
  },
  "peerDependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  },
  "dependencies": {
    "@headlessui/react": "^2.1.0",
    "lucide-react": "^0.446.0",
    "clsx": "^2.1.0",
    "tailwind-merge": "^2.5.0"
  },
  "devDependencies": {
    "tsup": "^8.3.0",
    "typescript": "^5.5.0"
  }
}
```

### Shared API Package
```tsx
// packages/api/src/index.ts
export * from './client'
export * from './auth'
export * from './bookings'
export * from './admin'
export * from './driver'
export * from './types'

// Re-export commonly used types
export type {
  User,
  Booking,
  Driver,
  Assignment,
  LoginRequest,
  CreateBookingRequest,
} from './types'
```

```tsx
// packages/api/src/driver.ts
import { apiClient } from './client'
import type { Assignment, Trip, DriverStatus, Earnings } from './types/driver'

export const driverAPI = {
  // Authentication
  login: (email: string, password: string) =>
    apiClient.post('/v1/auth/login', { email, password }),

  // Assignment management
  getAssignments: (): Promise<Assignment[]> =>
    apiClient.get('/v1/driver/assignments'),
    
  acceptAssignment: (assignmentId: number): Promise<Trip> =>
    apiClient.post(`/v1/driver/assignments/${assignmentId}/accept`),
    
  declineAssignment: (assignmentId: number, reason?: string): Promise<void> =>
    apiClient.post(`/v1/driver/assignments/${assignmentId}/decline`, { reason }),

  // Trip management
  startTrip: (tripId: number, location: { lat: number; lng: number }): Promise<Trip> =>
    apiClient.post(`/v1/driver/trips/${tripId}/start`, { location }),
    
  completeTrip: (tripId: number, location: { lat: number; lng: number }): Promise<Trip> =>
    apiClient.post(`/v1/driver/trips/${tripId}/complete`, { location }),
    
  updateLocation: (location: { lat: number; lng: number }): Promise<void> =>
    apiClient.post('/v1/driver/location', { location }),

  // Status management
  getStatus: (): Promise<DriverStatus> =>
    apiClient.get('/v1/driver/status'),
    
  setAvailability: (available: boolean): Promise<DriverStatus> =>
    apiClient.post('/v1/driver/availability', { available }),
    
  goOnline: (): Promise<DriverStatus> =>
    apiClient.post('/v1/driver/status/online'),
    
  goOffline: (): Promise<DriverStatus> =>
    apiClient.post('/v1/driver/status/offline'),

  // Earnings
  getEarnings: (period: 'today' | 'week' | 'month'): Promise<Earnings> =>
    apiClient.get(`/v1/driver/earnings?period=${period}`),
    
### Driver Mobile Navigation
```tsx
// apps/driver/src/components/mobile/MobileNav.tsx
import React from 'react'
import { useRouter } from '@tanstack/react-router'
import { Home, MapPin, Clock, DollarSign, User } from 'lucide-react'
import { clsx } from 'clsx'

const navItems = [
  { icon: Home, label: 'Home', path: '/' },
  { icon: MapPin, label: 'Trips', path: '/trips' },
  { icon: Clock, label: 'History', path: '/history' },
  { icon: DollarSign, label: 'Earnings', path: '/earnings' },
  { icon: User, label: 'Profile', path: '/profile' },
]

export const MobileNav: React.FC = () => {
  const router = useRouter()
  const currentPath = router.state.location.pathname

  return (
    <nav className="fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 safe-area-pb">
      <div className="grid grid-cols-5">
        {navItems.map(({ icon: Icon, label, path }) => {
          const isActive = currentPath === path
          
          return (
            <button
              key={path}
              onClick={() => router.navigate({ to: path })}
              className={clsx(
                'flex flex-col items-center justify-center py-2 px-1 transition-colors',
                isActive
                  ? 'text-primary-600 bg-primary-50'
                  : 'text-gray-600 hover:text-gray-900'
              )}
            >
              <Icon className="w-5 h-5 mb-1" />
              <span className="text-xs font-medium">{label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
```

## 🔧 App-Specific Configurations

### Rider App Environment
```env
# apps/rider/.env.example
VITE_API_URL=http://localhost:8080
VITE_APP_NAME=LuxSuv Rider
VITE_APP_TYPE=rider
VITE_ENVIRONMENT=development

# Rider-specific features
VITE_ENABLE_GUEST_BOOKING=true
VITE_ENABLE_PAYMENT=true
VITE_STRIPE_PUBLISHABLE_KEY=pk_test_...

# Analytics
VITE_GA_MEASUREMENT_ID=G-XXXXXXXXXX
VITE_HOTJAR_ID=
```

### Admin Portal Environment
```env
# apps/admin/.env.example
VITE_API_URL=http://localhost:8080
VITE_APP_NAME=LuxSuv Operations
VITE_APP_TYPE=admin
VITE_ENVIRONMENT=development

# Admin-specific features
VITE_ENABLE_ANALYTICS=true
VITE_ENABLE_REAL_TIME=true
VITE_MAP_API_KEY=
VITE_DASHBOARD_REFRESH_INTERVAL=30000

# Monitoring
VITE_SENTRY_DSN=
VITE_DATADOG_CLIENT_TOKEN=
```

### Driver App Environment
```env
# apps/driver/.env.example
VITE_API_URL=http://localhost:8080
VITE_APP_NAME=LuxSuv Driver
VITE_APP_TYPE=driver
VITE_ENVIRONMENT=development

# Driver-specific features
VITE_ENABLE_GPS=true
VITE_ENABLE_OFFLINE_MODE=true
VITE_LOCATION_UPDATE_INTERVAL=10000
VITE_MAP_API_KEY=

# Mobile features
VITE_ENABLE_PUSH_NOTIFICATIONS=true
VITE_ENABLE_VIBRATION=true
VITE_ENABLE_WAKE_LOCK=true
```

## 🎨 App-Specific Design Systems

### Rider App (Customer-Friendly)
```js
// apps/rider/tailwind.config.js
import { createTailwindConfig } from '@luxsuv/config/tailwind'

export default createTailwindConfig({
  theme: {
    extend: {
      colors: {
        primary: {
          // Warm, inviting blues for customers
          500: '#3b82f6',
          600: '#2563eb',
        },
        accent: {
          // Gold accents for luxury feel
          400: '#fbbf24',
          500: '#f59e0b',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
      borderRadius: {
        'xl': '1rem',
        '2xl': '1.5rem',
      },
    },
  },
})
```

### Admin Portal (Professional)
```js
// apps/admin/tailwind.config.js
import { createTailwindConfig } from '@luxsuv/config/tailwind'

export default createTailwindConfig({
  theme: {
    extend: {
      colors: {
        primary: {
          // Professional navy blues for admin
          500: '#1e40af',
          600: '#1d4ed8',
        },
        accent: {
          // Subtle accent colors for data viz
          500: '#8b5cf6',
          600: '#7c3aed',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'], // For data tables
      },
    },
  },
})
```

### Driver App (Mobile-Optimized)
```js
// apps/driver/tailwind.config.js
import { createTailwindConfig } from '@luxsuv/config/tailwind'

export default createTailwindConfig({
  theme: {
    extend: {
      colors: {
        primary: {
          // Vibrant greens for drivers (go/stop)
          500: '#10b981',
          600: '#059669',
        },
        accent: {
          // Orange for alerts and actions
          500: '#f97316',
          600: '#ea580c',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
      spacing: {
        'safe-top': 'env(safe-area-inset-top)',
        'safe-bottom': 'env(safe-area-inset-bottom)',
      },
      minHeight: {
        'screen-safe': 'calc(100vh - env(safe-area-inset-top) - env(safe-area-inset-bottom))',
      },
    },
  },
})
```

## 🚀 Development Workflow

### Start All Apps
```bash
# Start all apps in development mode
pnpm dev

# Or start individually
pnpm dev:rider     # http://localhost:3001
pnpm dev:admin     # http://localhost:3002  
pnpm dev:driver    # http://localhost:3003
```

### Build and Deploy
```bash
# Build all apps
pnpm build

# Build specific app
turbo run build --filter=rider

# Deploy to different environments
pnpm deploy:rider --env=staging
pnpm deploy:admin --env=production
pnpm deploy:driver --env=production
```

## 🎯 App-Specific Features

### 🚗 Rider App Features

#### Guest Booking Flow
```tsx
// apps/rider/src/routes/book.tsx
import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { ArrowRight, Clock, MapPin } from 'lucide-react'
import { Button, Input } from '@luxsuv/ui'
import { BookingForm } from '../components/forms/BookingForm'
import { GuestAccessModal } from '../components/modals/GuestAccessModal'

export const Route = createFileRoute('/book')({
  component: QuickBookingPage,
})

function QuickBookingPage() {
  const [showGuestModal, setShowGuestModal] = useState(false)
  const [bookingData, setBookingData] = useState(null)

  const handleGuestBooking = (data) => {
    setBookingData(data)
    setShowGuestModal(true)
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      {/* Hero Section */}
      <div className="container mx-auto px-4 pt-12 pb-8">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-gray-900 mb-4">
            Luxury Transportation
          </h1>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto">
            Book premium SUV rides with professional drivers. 
            No account required for quick bookings.
          </p>
        </div>

        {/* Quick Booking Card */}
        <div className="max-w-2xl mx-auto bg-white rounded-2xl shadow-xl p-8">
          <BookingForm
            onSubmit={handleGuestBooking}
            submitLabel="Book Now"
            showContactFields={true}
          />
        </div>
      </div>

      {/* Guest Access Modal */}
      <GuestAccessModal
        isOpen={showGuestModal}
        onClose={() => setShowGuestModal(false)}
        bookingData={bookingData}
      />
    </div>
  )
}
}
```

### Base UI Components

#### Button Component
```tsx
// src/components/ui/Button.tsx
import React from 'react'
import { clsx } from 'clsx'
import { Loader2 } from 'lucide-react'

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  isLoading?: boolean
  leftIcon?: React.ReactNode
  rightIcon?: React.ReactNode
}

export const Button: React.FC<ButtonProps> = ({
  variant = 'primary',
  size = 'md',
  isLoading = false,
  leftIcon,
  rightIcon,
  children,
  className,
  disabled,
  ...props
}) => {
  const baseClasses = 'inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed'
  
  const variants = {
    primary: 'bg-primary-600 text-white hover:bg-primary-700 focus:ring-primary-500 shadow-sm hover:shadow-md',
    secondary: 'bg-gray-100 text-gray-900 hover:bg-gray-200 focus:ring-gray-500 border border-gray-300',
    ghost: 'text-gray-700 hover:bg-gray-100 focus:ring-gray-500',
    danger: 'bg-error-600 text-white hover:bg-error-700 focus:ring-error-500 shadow-sm hover:shadow-md',
  }
  
  const sizes = {
    sm: 'px-3 py-1.5 text-sm gap-1.5',
    md: 'px-4 py-2 text-base gap-2',
    lg: 'px-6 py-3 text-lg gap-2.5',
  }

  return (
    <button
      className={clsx(
        baseClasses,
        variants[variant],
        sizes[size],
        className
      )}
      disabled={disabled || isLoading}
      {...props}
    >
      {isLoading ? (
        <Loader2 className="w-4 h-4 animate-spin" />
      ) : (
        leftIcon
      )}
      {children}
      {!isLoading && rightIcon}
    </button>
  )
}
```

#### Input Component
```tsx
// src/components/ui/Input.tsx
import React from 'react'
import { clsx } from 'clsx'

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  helpText?: string
  leftIcon?: React.ReactNode
  rightIcon?: React.ReactNode
}

export const Input: React.FC<InputProps> = ({
  label,
  error,
  helpText,
  leftIcon,
  rightIcon,
  className,
  id,
  ...props
}) => {
  const inputId = id || `input-${Math.random().toString(36).substr(2, 9)}`
  
  return (
    <div className="space-y-1">
      {label && (
        <label 
          htmlFor={inputId}
          className="block text-sm font-medium text-gray-700"
        >
          {label}
        </label>
      )}
      
      <div className="relative">
        {leftIcon && (
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <div className="h-5 w-5 text-gray-400">
              {leftIcon}
            </div>
          </div>
        )}
        
        <input
          id={inputId}
          className={clsx(
            'block w-full rounded-lg border-gray-300 shadow-sm transition-colors duration-200',
            'focus:border-primary-500 focus:ring-primary-500',
            'disabled:bg-gray-50 disabled:text-gray-500',
            {
              'border-error-300 focus:border-error-500 focus:ring-error-500': error,
              'pl-10': leftIcon,
              'pr-10': rightIcon,
            },
            className
          )}
          {...props}
        />
        
        {rightIcon && (
          <div className="absolute inset-y-0 right-0 pr-3 flex items-center">
            <div className="h-5 w-5 text-gray-400">
              {rightIcon}
            </div>
          </div>
        )}
      </div>
      
      {error && (
        <p className="text-sm text-error-600">{error}</p>
      )}
      
      {helpText && !error && (
        <p className="text-sm text-gray-500">{helpText}</p>
      )}
    </div>
  )
}
```

## 🔄 Real-time Features

### WebSocket Integration
```tsx
// packages/api/src/websocket.ts
import { useEffect, useRef, useState } from 'react'

interface UseWebSocketOptions {
  onMessage?: (message: any) => void
  onConnect?: () => void
  onDisconnect?: () => void
  reconnectAttempts?: number
  reconnectInterval?: number
}

export function useWebSocket(url: string, options: UseWebSocketOptions = {}) {
  const {
    onMessage,
    onConnect,
    onDisconnect,
    reconnectAttempts = 5,
    reconnectInterval = 5000,
  } = options

  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<any>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectCountRef = useRef(0)

  const connect = () => {
    const token = localStorage.getItem('auth_token') || localStorage.getItem('guest_session_token')
    const wsUrl = `${url}?token=${token}`
    
    try {
      wsRef.current = new WebSocket(wsUrl)
      
      wsRef.current.onopen = () => {
        setIsConnected(true)
        reconnectCountRef.current = 0
        onConnect?.()
      }
      
      wsRef.current.onmessage = (event) => {
        const message = JSON.parse(event.data)
        setLastMessage(message)
        onMessage?.(message)
      }
      
      wsRef.current.onclose = () => {
        setIsConnected(false)
        onDisconnect?.()
        
        // Auto-reconnect
        if (reconnectCountRef.current < reconnectAttempts) {
          setTimeout(() => {
            reconnectCountRef.current++
            connect()
          }, reconnectInterval)
        }
      }
      
      wsRef.current.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error)
    }
  }

  useEffect(() => {
    connect()
    
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [url])

  const sendMessage = (message: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
    }
  }

  return {
    isConnected,
    lastMessage,
    sendMessage,
    reconnect: connect,
  }
}
```

## 🎯 App-Specific Implementation Priorities

### 🚗 Rider App - Phase 1 (MVP)
- [ ] Landing page with quick booking
- [ ] Guest access flow (email → code → booking)
- [ ] Booking form with validation
- [ ] Booking status tracking
- [ ] User registration and login
- [ ] Authenticated user booking flow
- [ ] Booking history and management

### 🏢 Admin Portal - Phase 1 (Core Operations)
- [ ] Admin authentication and dashboard
- [ ] Booking management (view, assign, cancel)
- [ ] Driver management (view status, availability)
- [ ] Manual driver assignment interface
- [ ] Real-time operations view
- [ ] User management (riders, drivers)
- [ ] Basic analytics and reporting

### 📱 Driver App - Phase 1 (Essential)
- [ ] Driver authentication
- [ ] Online/offline status toggle
- [ ] Assignment notifications and acceptance
- [ ] Trip start/complete workflow
- [ ] Basic earnings display
- [ ] Location sharing and GPS
- [ ] Trip navigation interface

## 📋 Development Setup Checklist

### Initial Setup
- [ ] Initialize monorepo with pnpm workspaces
- [ ] Set up Turborepo for build orchestration
- [ ] Create shared packages (UI, API, Utils)
- [ ] Configure TypeScript for all packages
- [ ] Set up ESLint and Prettier configs
- [ ] Configure Tailwind for each app

### Rider App Setup
- [ ] Create Vite React app with TypeScript
- [ ] Install TanStack Router and Query
- [ ] Set up authentication store and hooks
- [ ] Implement guest access flow
- [ ] Create booking forms and validation
- [ ] Add responsive design for mobile

### Admin Portal Setup
- [ ] Create Vite React app with TypeScript
- [ ] Install dashboard and table dependencies
- [ ] Set up admin authentication
- [ ] Implement real-time dashboard
- [ ] Create data tables with filtering
- [ ] Add assignment management interface

### Driver App Setup
- [ ] Create Vite React app with TypeScript
- [ ] Configure PWA with service worker
- [ ] Set up mobile-first responsive design
- [ ] Implement geolocation and GPS
- [ ] Create assignment management interface
- [ ] Add trip tracking and navigation
- [ ] Configure push notifications

### Deployment Setup
- [ ] Configure Vercel for rider app
- [ ] Configure Netlify for admin portal
- [ ] Configure PWA hosting for driver app
- [ ] Set up CI/CD pipelines
- [ ] Configure environment variables
- [ ] Set up monitoring and analytics

## 🚀 Deployment URLs

### Development
- **Rider App**: http://localhost:3001
- **Admin Portal**: http://localhost:3002
- **Driver App**: http://localhost:3003
- **Shared UI Storybook**: http://localhost:6006

### Production
- **Rider App**: https://book.luxsuv.com
- **Admin Portal**: https://admin.luxsuv.com
- **Driver App**: https://driver.luxsuv.com
- **API Gateway**: https://api.luxsuv.com

This multi-app architecture provides:

✅ **Separation of Concerns** - Each app focuses on specific user needs
✅ **Independent Deployment** - Deploy apps separately without affecting others  
✅ **Optimized UX** - Tailored interfaces for each user type
✅ **Scalable Development** - Teams can work on different apps independently
✅ **Mobile-Ready** - Driver and rider apps optimized for mobile use
✅ **Shared Code** - Common components and logic in shared packages

Each app can be developed, tested, and deployed independently while sharing common functionality through the shared packages. This approach scales well as your team grows and provides the flexibility to migrate to React Native for mobile apps later.

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`
    const token = localStorage.getItem('auth_token') || localStorage.getItem('guest_session_token')
    
    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      },
      ...options,
    }

    try {
      const response = await fetch(url, config)
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new APIError(
          response.status,
          errorData.code,
          errorData.details,
          errorData.error || response.statusText
        )
      }

      // Handle no content responses
      if (response.status === 204) {
        return null as T
      }

      return await response.json()
    } catch (error) {
      if (error instanceof APIError) {
        throw error
      }
      throw new APIError(0, 'NETWORK_ERROR', undefined, 'Network request failed')
    }
  }

  // HTTP methods
  get<T>(endpoint: string, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET', ...options })
  }

  post<T>(endpoint: string, data?: any, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
      ...options,
    })
  }

  patch<T>(endpoint: string, data?: any, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PATCH', 
      body: data ? JSON.stringify(data) : undefined,
      ...options,
    })
  }

  delete<T>(endpoint: string, options?: RequestInit): Promise<T> {
    return this.request<T>(endpoint, { method: 'DELETE', ...options })
  }
}

export const apiClient = new APIClient()

// React Query setup
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        if (error instanceof APIError && error.status === 401) return false
        return failureCount < 3
      },
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000, // 10 minutes
    },
    mutations: {
      retry: false,
    },
  },
})
```

### Auth API Methods
```tsx
// src/lib/api/auth.ts
import { apiClient } from './client'
import type { LoginRequest, RegisterRequest, User, LoginResponse } from '../types/auth'

export const authAPI = {
  // User authentication
  register: (data: RegisterRequest): Promise<{ message: string; user: User; dev_verify_url?: string }> =>
    apiClient.post('/v1/auth/register', data),
    
  login: (data: LoginRequest): Promise<LoginResponse> =>
    apiClient.post('/v1/auth/login', data),
    
  verifyEmail: (token: string): Promise<{ message: string; user: User }> =>
    apiClient.post(`/v1/auth/verify-email?token=${token}`),
    
  resendVerification: (email: string): Promise<{ message: string }> =>
    apiClient.post('/v1/auth/resend-verification', { email }),
    
  refreshToken: (refreshToken: string): Promise<LoginResponse> =>
    apiClient.post('/v1/auth/refresh', { refresh_token: refreshToken }),

  // Guest access
  requestGuestAccess: (email: string): Promise<{ message: string }> =>
    apiClient.post('/v1/guest/access/request', { email }),
    
  verifyGuestCode: (email: string, code: string): Promise<{ session_token: string; expires_in: number }> =>
    apiClient.post('/v1/guest/access/verify', { email, code }),
    
  verifyMagicLink: (token: string): Promise<{ session_token: string; expires_in: number }> =>
    apiClient.post(`/v1/guest/access/magic?token=${token}`),
}
```

### Bookings API Methods
```tsx
// src/lib/api/bookings.ts
import { apiClient } from './client'
import type { CreateBookingRequest, Booking, BookingListResponse, GuestBookingResponse } from '../types/booking'

export const bookingsAPI = {
  // Guest bookings
  createGuestBooking: (data: CreateBookingRequest, idempotencyKey?: string): Promise<GuestBookingResponse> =>
    apiClient.post('/v1/guest/bookings', data, {
      headers: idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : {},
    }),
    
  listGuestBookings: (params: { limit?: number; offset?: number; status?: string } = {}): Promise<BookingListResponse> => {
    const searchParams = new URLSearchParams()
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) searchParams.append(key, String(value))
    })
    return apiClient.get(`/v1/guest/bookings?${searchParams}`)
  },
  
  getGuestBooking: (id: number, manageToken?: string): Promise<Booking> => {
    const url = manageToken 
      ? `/v1/guest/bookings/${id}?manage_token=${manageToken}`
      : `/v1/guest/bookings/${id}`
    return apiClient.get(url)
  },
  
  updateGuestBooking: (id: number, data: Partial<CreateBookingRequest>, manageToken?: string): Promise<Booking> => {
    const url = manageToken
      ? `/v1/guest/bookings/${id}?manage_token=${manageToken}`
      : `/v1/guest/bookings/${id}`
    return apiClient.patch(url, data)
  },
  
  cancelGuestBooking: (id: number, manageToken?: string): Promise<void> => {
    const url = manageToken
      ? `/v1/guest/bookings/${id}?manage_token=${manageToken}`
      : `/v1/guest/bookings/${id}`
    return apiClient.delete(url)
  },

  // Rider bookings (authenticated)
  createRiderBooking: (data: Omit<CreateBookingRequest, 'rider_name' | 'rider_email' | 'rider_phone'>): Promise<Booking> =>
    apiClient.post('/v1/rider/bookings', data),
    
  listRiderBookings: (params: { limit?: number; offset?: number; status?: string } = {}): Promise<BookingListResponse> => {
    const searchParams = new URLSearchParams()
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) searchParams.append(key, String(value))
    })
    return apiClient.get(`/v1/rider/bookings?${searchParams}`)
  },
  
  getRiderBooking: (id: number): Promise<Booking> =>
    apiClient.get(`/v1/rider/bookings/${id}`),
    
  updateRiderBooking: (id: number, data: Partial<CreateBookingRequest>): Promise<Booking> =>
    apiClient.patch(`/v1/rider/bookings/${id}`, data),
    
  cancelRiderBooking: (id: number): Promise<void> =>
    apiClient.delete(`/v1/rider/bookings/${id}`),
}
```

## ⚛️ State Management

### Authentication Store (Zustand)
```tsx
// src/stores/auth.ts
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface User {
  id: number
  email: string
  name: string
  phone: string
  role: string
  is_verified: boolean
}

interface AuthState {
  // User state
  user: User | null
  isAuthenticated: boolean
  
  // Guest state
  guestEmail: string | null
  guestSessionToken: string | null
  guestExpiresAt: number | null
  
  // Actions
  setUser: (user: User, accessToken: string, refreshToken?: string) => void
  setGuestSession: (email: string, token: string, expiresIn: number) => void
  clearAuth: () => void
  clearGuestSession: () => void
  
  // Getters
  isGuestAuthenticated: () => boolean
  isTokenExpired: () => boolean
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      isAuthenticated: false,
      guestEmail: null,
      guestSessionToken: null,
      guestExpiresAt: null,
      
      // Actions
      setUser: (user, accessToken, refreshToken) => {
        localStorage.setItem('auth_token', accessToken)
        if (refreshToken) {
          localStorage.setItem('refresh_token', refreshToken)
        }
        set({ user, isAuthenticated: true })
      },
      
      setGuestSession: (email, token, expiresIn) => {
        const expiresAt = Date.now() + (expiresIn * 1000)
        localStorage.setItem('guest_session_token', token)
        set({
          guestEmail: email,
          guestSessionToken: token,
          guestExpiresAt: expiresAt,
        })
      },
      
      clearAuth: () => {
        localStorage.removeItem('auth_token')
        localStorage.removeItem('refresh_token')
        set({ user: null, isAuthenticated: false })
      },
      
      clearGuestSession: () => {
        localStorage.removeItem('guest_session_token')
        set({
          guestEmail: null,
          guestSessionToken: null,
          guestExpiresAt: null,
        })
      },
      
      // Getters
      isGuestAuthenticated: () => {
        const state = get()
        return !!(
          state.guestSessionToken &&
          state.guestExpiresAt &&
          Date.now() < state.guestExpiresAt
        )
      },
      
      isTokenExpired: () => {
        const state = get()
        return !!(
          state.guestExpiresAt &&
          Date.now() >= state.guestExpiresAt
        )
      },
    }),
    {
      name: 'luxsuv-auth',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        guestEmail: state.guestEmail,
        guestExpiresAt: state.guestExpiresAt,
      }),
    }
  )
)
```

### Authentication Hooks
```tsx
// src/hooks/api/useAuth.ts
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { authAPI } from '../../lib/api/auth'
import { useAuthStore } from '../../stores/auth'
import { useToast } from '../ui/useToast'
import type { LoginRequest, RegisterRequest } from '../../types/auth'

export function useAuth() {
  const { user, isAuthenticated, setUser, clearAuth } = useAuthStore()
  const navigate = useNavigate()
  const { toast } = useToast()

  // User registration
  const registerMutation = useMutation({
    mutationFn: (data: RegisterRequest) => authAPI.register(data),
    onSuccess: (data) => {
      toast.success('Registration successful! Please check your email to verify your account.')
      if (data.dev_verify_url) {
        console.log('Dev verification URL:', data.dev_verify_url)
      }
      navigate({ to: '/auth/login' })
    },
    onError: (error: any) => {
      toast.error(error.message || 'Registration failed')
    },
  })

  // User login
  const loginMutation = useMutation({
    mutationFn: (data: LoginRequest) => authAPI.login(data),
    onSuccess: (data) => {
      setUser(data.user, data.access_token, data.refresh_token)
      toast.success('Welcome back!')
      navigate({ to: '/dashboard' })
    },
    onError: (error: any) => {
      toast.error(error.message || 'Login failed')
    },
  })

  // Email verification
  const verifyEmailMutation = useMutation({
    mutationFn: (token: string) => authAPI.verifyEmail(token),
    onSuccess: (data) => {
      toast.success('Email verified successfully!')
      navigate({ to: '/auth/login' })
    },
    onError: (error: any) => {
      toast.error(error.message || 'Email verification failed')
    },
  })

  // Logout
  const logout = () => {
    clearAuth()
    navigate({ to: '/' })
    toast.success('Logged out successfully')
  }

  return {
    user,
    isAuthenticated,
    register: registerMutation.mutate,
    isRegistering: registerMutation.isPending,
    login: loginMutation.mutate,
    isLoggingIn: loginMutation.isPending,
    verifyEmail: verifyEmailMutation.mutate,
    isVerifyingEmail: verifyEmailMutation.isPending,
    logout,
  }
}
```

### Guest Access Hooks
```tsx
// src/hooks/api/useGuestAccess.ts
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { authAPI } from '../../lib/api/auth'
import { useAuthStore } from '../../stores/auth'
import { useToast } from '../ui/useToast'

export function useGuestAccess() {
  const { setGuestSession, guestEmail } = useAuthStore()
  const navigate = useNavigate()
  const { toast } = useToast()

  // Request access code
  const requestAccessMutation = useMutation({
    mutationFn: (email: string) => authAPI.requestGuestAccess(email),
    onSuccess: () => {
      toast.success('Access code sent to your email!')
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to send access code')
    },
  })

  // Verify access code
  const verifyCodeMutation = useMutation({
    mutationFn: ({ email, code }: { email: string; code: string }) =>
      authAPI.verifyGuestCode(email, code),
    onSuccess: (data, variables) => {
      setGuestSession(variables.email, data.session_token, data.expires_in)
      toast.success('Access granted!')
      navigate({ to: '/guest/bookings' })
    },
    onError: (error: any) => {
      toast.error(error.message || 'Invalid or expired code')
    },
  })

  // Verify magic link
  const verifyMagicMutation = useMutation({
    mutationFn: (token: string) => authAPI.verifyMagicLink(token),
    onSuccess: (data) => {
      // Note: We don't know the email from magic link response
      // The backend should include it in the response
      toast.success('Access granted via magic link!')
      navigate({ to: '/guest/bookings' })
    },
    onError: (error: any) => {
      toast.error(error.message || 'Invalid or expired magic link')
    },
  })

  return {
    guestEmail,
    requestAccess: requestAccessMutation.mutate,
    isRequestingAccess: requestAccessMutation.isPending,
    verifyCode: verifyCodeMutation.mutate,
    isVerifyingCode: verifyCodeMutation.isPending,
    verifyMagicLink: verifyMagicMutation.mutate,
    isVerifyingMagic: verifyMagicMutation.isPending,
  }
}
```

### Bookings Hooks
```tsx
// src/hooks/api/useBookings.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { bookingsAPI } from '../../lib/api/bookings'
import { useAuthStore } from '../../stores/auth'
import { useToast } from '../ui/useToast'
import type { CreateBookingRequest, Booking } from '../../types/booking'

export function useBookings() {
  const { isAuthenticated, isGuestAuthenticated } = useAuthStore()
  const queryClient = useQueryClient()
  const { toast } = useToast()
  
  const isAnyAuth = isAuthenticated || isGuestAuthenticated()

  // List bookings
  const bookingsQuery = useQuery({
    queryKey: ['bookings'],
    queryFn: () => isAuthenticated 
      ? bookingsAPI.listRiderBookings()
      : bookingsAPI.listGuestBookings(),
    enabled: isAnyAuth,
  })

  // Create booking mutation
  const createBookingMutation = useMutation({
    mutationFn: ({ data, idempotencyKey }: { data: CreateBookingRequest; idempotencyKey?: string }) =>
      isAuthenticated 
        ? bookingsAPI.createRiderBooking(data)
        : bookingsAPI.createGuestBooking(data, idempotencyKey),
    onSuccess: (newBooking) => {
      queryClient.setQueryData(['bookings'], (old: Booking[] = []) => 
        [newBooking, ...old]
      )
      toast.success('Booking created successfully!')
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to create booking')
    },
  })

  // Update booking mutation
  const updateBookingMutation = useMutation({
    mutationFn: ({ id, data, manageToken }: { id: number; data: Partial<CreateBookingRequest>; manageToken?: string }) =>
      isAuthenticated
        ? bookingsAPI.updateRiderBooking(id, data)
        : bookingsAPI.updateGuestBooking(id, data, manageToken),
    onSuccess: (updatedBooking) => {
      queryClient.setQueryData(['bookings'], (old: Booking[] = []) =>
        old.map(booking => booking.id === updatedBooking.id ? updatedBooking : booking)
      )
      queryClient.setQueryData(['booking', updatedBooking.id], updatedBooking)
      toast.success('Booking updated successfully!')
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to update booking')
    },
  })

  // Cancel booking mutation
  const cancelBookingMutation = useMutation({
    mutationFn: ({ id, manageToken }: { id: number; manageToken?: string }) =>
      isAuthenticated
        ? bookingsAPI.cancelRiderBooking(id)
        : bookingsAPI.cancelGuestBooking(id, manageToken),
    onSuccess: (_, variables) => {
      queryClient.setQueryData(['bookings'], (old: Booking[] = []) =>
        old.map(booking => 
          booking.id === variables.id 
            ? { ...booking, status: 'canceled' }
            : booking
        )
      )
      toast.success('Booking canceled successfully')
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to cancel booking')
    },
  })

  return {
    // Data
    bookings: bookingsQuery.data || [],
    isLoading: bookingsQuery.isLoading,
    isError: bookingsQuery.isError,
    error: bookingsQuery.error,
    
    // Actions
    createBooking: createBookingMutation.mutate,
    isCreating: createBookingMutation.isPending,
    updateBooking: updateBookingMutation.mutate,
    isUpdating: updateBookingMutation.isPending,
    cancelBooking: cancelBookingMutation.mutate,
    isCanceling: cancelBookingMutation.isPending,
    
    // Utils
    refetch: bookingsQuery.refetch,
  }
}

// Hook for single booking
export function useBooking(id: number, manageToken?: string) {
  const { isAuthenticated } = useAuthStore()
  
  return useQuery({
    queryKey: ['booking', id, manageToken],
    queryFn: () => isAuthenticated
      ? bookingsAPI.getRiderBooking(id)
      : bookingsAPI.getGuestBooking(id, manageToken),
    enabled: !!id,
  })
}
```

## 🗂️ Form Management

### Booking Form Component
```tsx
// src/components/forms/BookingForm.tsx
import React from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { format } from 'date-fns'
import { Calendar, MapPin, Users, Luggage } from 'lucide-react'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { createBookingSchema } from '../../lib/validations/booking'
import { useAuthStore } from '../../stores/auth'
import type { CreateBookingRequest } from '../../types/booking'

interface BookingFormProps {
  defaultValues?: Partial<CreateBookingRequest>
  onSubmit: (data: CreateBookingRequest) => void
  isLoading?: boolean
  submitLabel?: string
}

export const BookingForm: React.FC<BookingFormProps> = ({
  defaultValues,
  onSubmit,
  isLoading = false,
  submitLabel = 'Create Booking',
}) => {
  const { isAuthenticated } = useAuthStore()
  
  const {
    register,
    handleSubmit,
    formState: { errors },
    watch,
    setValue,
  } = useForm<CreateBookingRequest>({
    resolver: zodResolver(createBookingSchema),
    defaultValues: {
      passengers: 1,
      luggages: 0,
      ride_type: 'per_ride',
      ...defaultValues,
    },
  })

  const watchedPassengers = watch('passengers')
  const watchedScheduledAt = watch('scheduled_at')

  // Auto-set minimum scheduled time (1 hour from now)
  React.useEffect(() => {
    if (!watchedScheduledAt) {
      const minTime = new Date()
      minTime.setHours(minTime.getHours() + 1)
      setValue('scheduled_at', format(minTime, "yyyy-MM-dd'T'HH:mm"))
    }
  }, [setValue, watchedScheduledAt])

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {/* Contact Information (only for guest bookings) */}
      {!isAuthenticated && (
        <div className="space-y-4">
          <h3 className="text-lg font-semibold text-gray-900">Contact Information</h3>
          
          <Input
            label="Full Name"
            {...register('rider_name')}
            error={errors.rider_name?.message}
            placeholder="John Doe"
          />
          
          <Input
            label="Email Address"
            type="email"
            {...register('rider_email')}
            error={errors.rider_email?.message}
            placeholder="john@example.com"
          />
          
          <Input
            label="Phone Number"
            type="tel"
            {...register('rider_phone')}
            error={errors.rider_phone?.message}
            placeholder="+1 (555) 123-4567"
          />
        </div>
      )}

      {/* Trip Details */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">Trip Details</h3>
        
        <Input
          label="Pickup Location"
          {...register('pickup')}
          error={errors.pickup?.message}
          placeholder="SFO Terminal 1"
          leftIcon={<MapPin />}
        />
        
        <Input
          label="Dropoff Location"
          {...register('dropoff')}
          error={errors.dropoff?.message}
          placeholder="Downtown Hotel"
          leftIcon={<MapPin />}
        />
        
        <Input
          label="Scheduled Time"
          type="datetime-local"
          {...register('scheduled_at')}
          error={errors.scheduled_at?.message}
          leftIcon={<Calendar />}
          min={format(new Date(), "yyyy-MM-dd'T'HH:mm")}
        />
        
        <div className="grid grid-cols-2 gap-4">
          <Input
            label="Passengers"
            type="number"
            {...register('passengers', { valueAsNumber: true })}
            error={errors.passengers?.message}
            min={1}
            max={8}
            leftIcon={<Users />}
          />
          
          <Input
            label="Luggage Count"
            type="number"
            {...register('luggages', { valueAsNumber: true })}
            error={errors.luggages?.message}
            min={0}
            max={10}
            leftIcon={<Luggage />}
          />
        </div>
        
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Ride Type
          </label>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="radio"
                {...register('ride_type')}
                value="per_ride"
                className="mr-3 text-primary-600 focus:ring-primary-500"
              />
              <span className="text-sm text-gray-700">Per Ride (Point to Point)</span>
            </label>
            <label className="flex items-center">
              <input
                type="radio"
                {...register('ride_type')}
                value="hourly"
                className="mr-3 text-primary-600 focus:ring-primary-500"
              />
              <span className="text-sm text-gray-700">Hourly Rate</span>
            </label>
          </div>
          {errors.ride_type && (
            <p className="mt-1 text-sm text-error-600">{errors.ride_type.message}</p>
          )}
        </div>
        
        <Input
          label="Special Notes"
          {...register('notes')}
          error={errors.notes?.message}
          placeholder="Any special requests or notes..."
          as="textarea"
          rows={3}
        />
      </div>

      {/* Submit Button */}
      <Button
        type="submit"
        isLoading={isLoading}
        className="w-full"
        size="lg"
      >
        {submitLabel}
      </Button>
    </form>
  )
}
```

## 🛣️ Routing Implementation

### Root Layout
```tsx
// src/routes/__root.tsx
import { createRootRoute, Outlet } from '@tanstack/react-router'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { Toaster } from 'sonner'
import { queryClient } from '../lib/api/client'
import { TanStackRouterDevtools } from '@tanstack/router-devtools'

export const Route = createRootRoute({
  component: RootComponent,
})

function RootComponent() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="min-h-screen bg-gray-50">
        <Outlet />
        <Toaster 
          position="top-right" 
          expand={false}
          richColors
          closeButton
        />
      </div>
      <ReactQueryDevtools initialIsOpen={false} />
      <TanStackRouterDevtools />
    </QueryClientProvider>
  )
}
```

### Guest Access Route
```tsx
// src/routes/guest/access.tsx
import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { Mail, KeyRound } from 'lucide-react'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { useGuestAccess } from '../../hooks/api/useGuestAccess'

export const Route = createFileRoute('/guest/access')({
  component: GuestAccessPage,
})

function GuestAccessPage() {
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [step, setStep] = useState<'request' | 'verify'>('request')
  
  const {
    requestAccess,
    isRequestingAccess,
    verifyCode,
    isVerifyingCode,
  } = useGuestAccess()

  const handleRequestAccess = () => {
    requestAccess(email, {
      onSuccess: () => {
        setStep('verify')
      },
    })
  }

  const handleVerifyCode = () => {
    verifyCode({ email, code })
  }

  if (step === 'verify') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary-50 to-primary-100">
        <div className="max-w-md w-full space-y-8 p-8">
          <div className="text-center">
            <KeyRound className="mx-auto h-12 w-12 text-primary-600" />
            <h2 className="mt-6 text-3xl font-bold text-gray-900">
              Enter Access Code
            </h2>
            <p className="mt-2 text-sm text-gray-600">
              We sent a 6-digit code to <strong>{email}</strong>
            </p>
          </div>
          
          <div className="space-y-4">
            <Input
              label="6-Digit Code"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="123456"
              maxLength={6}
              className="text-center text-2xl tracking-widest"
            />
            
            <Button
              onClick={handleVerifyCode}
              isLoading={isVerifyingCode}
              disabled={code.length !== 6}
              className="w-full"
            >
              Verify Code
            </Button>
            
            <Button
              variant="ghost"
              onClick={() => setStep('request')}
              className="w-full"
            >
              Use Different Email
            </Button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary-50 to-primary-100">
      <div className="max-w-md w-full space-y-8 p-8">
        <div className="text-center">
          <Mail className="mx-auto h-12 w-12 text-primary-600" />
          <h2 className="mt-6 text-3xl font-bold text-gray-900">
            Quick Access
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            Enter your email to receive an instant access code
          </p>
        </div>
        
        <div className="space-y-4">
          <Input
            label="Email Address"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="john@example.com"
            leftIcon={<Mail />}
          />
          
          <Button
            onClick={handleRequestAccess}
            isLoading={isRequestingAccess}
            disabled={!email}
            className="w-full"
          >
            Send Access Code
          </Button>
        </div>
      </div>
    </div>
  )
}
```

### Guest Bookings Route
```tsx
// src/routes/guest/bookings/index.tsx
import { createFileRoute } from '@tanstack/react-router'
import { Plus, Calendar, MapPin, Users, Clock } from 'lucide-react'
import { format } from 'date-fns'
import { Button } from '../../../components/ui/Button'
import { useBookings } from '../../../hooks/api/useBookings'
import { useAuthStore } from '../../../stores/auth'
import type { Booking } from '../../../types/booking'

export const Route = createFileRoute('/guest/bookings/')({
  component: GuestBookingsPage,
  beforeLoad: ({ context }) => {
    const { isGuestAuthenticated } = useAuthStore.getState()
    if (!isGuestAuthenticated()) {
      throw redirect({ to: '/guest/access' })
    }
  },
})

function GuestBookingsPage() {
  const { bookings, isLoading, isError } = useBookings()
  const navigate = useNavigate()

  if (isLoading) {
    return <BookingListSkeleton />
  }

  if (isError) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="text-center py-12">
          <p className="text-gray-500">Failed to load bookings. Please try again.</p>
          <Button onClick={() => window.location.reload()} className="mt-4">
            Retry
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold text-gray-900">My Bookings</h1>
        <Button
          onClick={() => navigate({ to: '/guest/bookings/create' })}
          leftIcon={<Plus />}
        >
          New Booking
        </Button>
      </div>

      {bookings.length === 0 ? (
        <EmptyBookingsState />
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {bookings.map((booking) => (
            <BookingCard key={booking.id} booking={booking} />
          ))}
        </div>
      )}
    </div>
  )
}

const BookingCard: React.FC<{ booking: Booking }> = ({ booking }) => {
  const navigate = useNavigate()
  
  const statusColors = {
    pending: 'bg-yellow-100 text-yellow-800',
    confirmed: 'bg-blue-100 text-blue-800',
    assigned: 'bg-purple-100 text-purple-800',
    on_trip: 'bg-green-100 text-green-800',
    completed: 'bg-gray-100 text-gray-800',
    canceled: 'bg-red-100 text-red-800',
  }

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow duration-200">
      <div className="flex justify-between items-start mb-4">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <MapPin className="w-4 h-4 text-gray-400" />
            <p className="text-sm text-gray-600">{booking.pickup}</p>
          </div>
          <div className="flex items-center gap-2">
            <MapPin className="w-4 h-4 text-gray-400" />
            <p className="text-sm text-gray-600">{booking.dropoff}</p>
          </div>
        </div>
        <span className={`px-2 py-1 rounded-full text-xs font-medium ${statusColors[booking.status]}`}>
          {booking.status.replace('_', ' ').toUpperCase()}
        </span>
      </div>
      
      <div className="space-y-2 mb-4">
        <div className="flex items-center gap-2 text-sm text-gray-600">
          <Calendar className="w-4 h-4" />
          {format(new Date(booking.scheduled_at), 'MMM d, yyyy h:mm a')}
        </div>
        <div className="flex items-center gap-2 text-sm text-gray-600">
          <Users className="w-4 h-4" />
          {booking.passengers} passenger{booking.passengers !== 1 ? 's' : ''}
          {booking.luggages > 0 && ` • ${booking.luggages} bag${booking.luggages !== 1 ? 's' : ''}`}
        </div>
      </div>
      
      {booking.notes && (
        <p className="text-sm text-gray-600 mb-4 bg-gray-50 p-3 rounded-lg">
          {booking.notes}
        </p>
      )}
      
      <Button
        onClick={() => navigate({ to: `/guest/bookings/${booking.id}` })}
        variant="secondary"
        size="sm"
        className="w-full"
      >
        View Details
      </Button>
    </div>
  )
}

const EmptyBookingsState = () => (
  <div className="text-center py-12">
    <Calendar className="mx-auto h-24 w-24 text-gray-300" />
    <h3 className="mt-4 text-lg font-semibold text-gray-900">No bookings yet</h3>
    <p className="mt-2 text-gray-500">Create your first luxury SUV booking to get started.</p>
    <Button
      onClick={() => navigate({ to: '/guest/bookings/create' })}
      className="mt-6"
      leftIcon={<Plus />}
    >
      Create Booking
    </Button>
  </div>
)

const BookingListSkeleton = () => (
  <div className="container mx-auto px-4 py-8">
    <div className="h-8 bg-gray-200 rounded w-48 mb-8 animate-pulse" />
    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
      {[...Array(6)].map((_, i) => (
        <div key={i} className="bg-white rounded-xl p-6 space-y-4">
          <div className="h-4 bg-gray-200 rounded animate-pulse" />
          <div className="h-4 bg-gray-200 rounded w-3/4 animate-pulse" />
          <div className="h-8 bg-gray-200 rounded animate-pulse" />
        </div>
      ))}
    </div>
  </div>
)
```

## 🔐 Authentication Flow Implementation

### Login Component
```tsx
// src/routes/auth/login.tsx
import { createFileRoute, Link } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Mail, Lock, ArrowRight } from 'lucide-react'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { useAuth } from '../../hooks/api/useAuth'
import { loginSchema } from '../../lib/validations/auth'
import type { LoginRequest } from '../../types/auth'

export const Route = createFileRoute('/auth/login')({
  component: LoginPage,
})

function LoginPage() {
  const { login, isLoggingIn } = useAuth()
  
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginRequest>({
    resolver: zodResolver(loginSchema),
  })

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary-50 to-primary-100">
      <div className="max-w-md w-full space-y-8 p-8 bg-white rounded-2xl shadow-xl">
        <div className="text-center">
          <h2 className="text-3xl font-bold text-gray-900">Welcome Back</h2>
          <p className="mt-2 text-sm text-gray-600">
            Sign in to your LuxSuv account
          </p>
        </div>
        
        <form onSubmit={handleSubmit(login)} className="space-y-6">
          <Input
            label="Email Address"
            type="email"
            {...register('email')}
            error={errors.email?.message}
            placeholder="john@example.com"
            leftIcon={<Mail />}
          />
          
          <Input
            label="Password"
            type="password"
            {...register('password')}
            error={errors.password?.message}
            placeholder="••••••••"
            leftIcon={<Lock />}
          />
          
          <Button
            type="submit"
            isLoading={isLoggingIn}
            className="w-full"
            rightIcon={<ArrowRight />}
          >
            Sign In
          </Button>
        </form>
        
        <div className="text-center space-y-4">
          <p className="text-sm text-gray-600">
            Don't have an account?{' '}
            <Link
              to="/auth/register"
              className="font-medium text-primary-600 hover:text-primary-500"
            >
              Sign up
            </Link>
          </p>
          
          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-gray-300" />
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="px-2 bg-white text-gray-500">Or</span>
            </div>
          </div>
          
          <Link to="/guest/access">
            <Button variant="secondary" className="w-full">
              Quick Guest Access
            </Button>
          </Link>
        </div>
      </div>
    </div>
  )
}
```

## 📱 Responsive Design Guidelines

### Breakpoint Strategy
```css
/* Mobile First Approach */
.container {
  @apply px-4 mx-auto;
  
  /* sm: 640px */
  @screen sm {
    @apply px-6;
  }
  
  /* md: 768px */
  @screen md {
    @apply px-8;
  }
  
  /* lg: 1024px */
  @screen lg {
    @apply px-12 max-w-7xl;
  }
}
```

### Component Responsive Patterns
```tsx
// Example: Responsive Booking Grid
<div className="grid gap-4 sm:gap-6 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
  {bookings.map(booking => (
    <BookingCard key={booking.id} booking={booking} />
  ))}
</div>

// Example: Responsive Form Layout  
<div className="grid gap-4 sm:grid-cols-2">
  <Input label="Passengers" />
  <Input label="Luggage" />
</div>

// Example: Mobile Navigation
<nav className="lg:hidden fixed bottom-0 left-0 right-0 bg-white border-t">
  <div className="grid grid-cols-4 py-2">
    {navItems.map(item => (
      <NavItem key={item.name} {...item} />
    ))}
  </div>
</nav>
```

## 🧪 Testing Strategy

### Unit Tests with Vitest
```tsx
// src/hooks/__tests__/useBookings.test.ts
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useBookings } from '../useBookings'
import { bookingsAPI } from '../../lib/api/bookings'

// Mock API
vi.mock('../../lib/api/bookings')
const mockBookingsAPI = vi.mocked(bookingsAPI)

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  )
}

describe('useBookings', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetches bookings successfully', async () => {
    const mockBookings = [
      { id: 1, rider_name: 'John Doe', status: 'pending' },
      { id: 2, rider_name: 'Jane Smith', status: 'confirmed' },
    ]
    
    mockBookingsAPI.listGuestBookings.mockResolvedValue({
      bookings: mockBookings,
      total: 2,
    })

    const { result } = renderHook(() => useBookings(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.bookings).toHaveLength(2)
      expect(result.current.isLoading).toBe(false)
    })
  })

  it('handles booking creation', async () => {
    const newBooking = { id: 3, rider_name: 'Bob Wilson', status: 'pending' }
    mockBookingsAPI.createGuestBooking.mockResolvedValue({
      id: 3,
      manage_token: 'token-123',
      status: 'pending',
      scheduled_at: new Date().toISOString(),
    })

    const { result } = renderHook(() => useBookings(), {
      wrapper: createWrapper(),
    })

    act(() => {
      result.current.createBooking({
        data: {
          rider_name: 'Bob Wilson',
          rider_email: 'bob@example.com',
          rider_phone: '+1234567890',
          pickup: 'Airport',
          dropoff: 'Hotel',
          scheduled_at: new Date().toISOString(),
          passengers: 1,
          luggages: 0,
          ride_type: 'per_ride',
        },
      })
    })

    await waitFor(() => {
      expect(result.current.isCreating).toBe(false)
    })
  })
})
```

### E2E Tests with Playwright
```tsx
// tests/booking-flow.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Guest Booking Flow', () => {
  test('complete guest booking journey', async ({ page }) => {
    // Navigate to guest access
    await page.goto('/guest/access')
    
    // Request access code
    await page.fill('[data-testid="email-input"]', 'test@example.com')
    await page.click('[data-testid="request-access-btn"]')
    
    // Verify code (in real test, you'd get this from test email)
    await page.fill('[data-testid="code-input"]', '123456')
    await page.click('[data-testid="verify-code-btn"]')
    
    // Should navigate to bookings page
    await expect(page).toHaveURL('/guest/bookings')
    
    // Create new booking
    await page.click('[data-testid="new-booking-btn"]')
    
    // Fill booking form
    await page.fill('[data-testid="pickup-input"]', 'SFO Terminal 1')
    await page.fill('[data-testid="dropoff-input"]', 'Downtown Hotel')
    await page.fill('[data-testid="passengers-input"]', '2')
    
    // Submit booking
    await page.click('[data-testid="create-booking-btn"]')
    
    // Verify booking created
    await expect(page.locator('[data-testid="booking-card"]')).toBeVisible()
    await expect(page.locator('text=SFO Terminal 1')).toBeVisible()
  })

  test('booking validation errors', async ({ page }) => {
    await page.goto('/guest/bookings/create')
    
    // Try to submit empty form
    await page.click('[data-testid="create-booking-btn"]')
    
    // Check for validation errors
    await expect(page.locator('text=Name is required')).toBeVisible()
    await expect(page.locator('text=Email is required')).toBeVisible()
    await expect(page.locator('text=Pickup location is required')).toBeVisible()
  })
})
```

## 🎯 Error Handling

### Error Boundary Component
```tsx
// src/components/ErrorBoundary.tsx
import React from 'react'
import { QueryErrorResetBoundary } from '@tanstack/react-query'
import { ErrorBoundary } from 'react-error-boundary'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { Button } from './ui/Button'

interface ErrorFallbackProps {
  error: Error
  resetErrorBoundary: () => void
}

const ErrorFallback: React.FC<ErrorFallbackProps> = ({ error, resetErrorBoundary }) => (
  <div className="min-h-screen flex items-center justify-center bg-gray-50">
    <div className="max-w-md w-full text-center p-8">
      <AlertTriangle className="mx-auto h-16 w-16 text-error-500 mb-6" />
      <h2 className="text-2xl font-bold text-gray-900 mb-4">Something went wrong</h2>
      <p className="text-gray-600 mb-6">
        {error.message || 'An unexpected error occurred. Please try again.'}
      </p>
      <Button
        onClick={resetErrorBoundary}
        leftIcon={<RefreshCw />}
      >
        Try Again
      </Button>
    </div>
  </div>
)

export const AppErrorBoundary: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <QueryErrorResetBoundary>
    {({ reset }) => (
      <ErrorBoundary onReset={reset} FallbackComponent={ErrorFallback}>
        {children}
      </ErrorBoundary>
    )}
  </QueryErrorResetBoundary>
)
```

### API Error Handling
```tsx
// src/lib/utils/errorHandling.ts
import { APIError } from '../api/client'

export const getErrorMessage = (error: unknown): string => {
  if (error instanceof APIError) {
    switch (error.code) {
      case 'INVALID_INPUT':
        return error.details || 'Please check your input and try again'
      case 'RATE_LIMIT_EXCEEDED':
        return 'Too many requests. Please wait a moment and try again'
      case 'UNAUTHORIZED':
        return 'Your session has expired. Please sign in again'
      case 'PAST_DATETIME':
        return 'Scheduled time must be in the future'
      default:
        return error.message || 'An unexpected error occurred'
    }
  }
  
  if (error instanceof Error) {
    return error.message
  }
  
  return 'An unexpected error occurred'
}

export const handleAPIError = (error: unknown, defaultMessage?: string) => {
  const message = getErrorMessage(error)
  console.error('API Error:', error)
  return defaultMessage || message
}
```

## 🎨 Animation and Micro-interactions

### Loading States
```tsx
// src/components/ui/LoadingSpinner.tsx
import { Loader2 } from 'lucide-react'
import { clsx } from 'clsx'

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({ 
  size = 'md', 
  className 
}) => {
  const sizes = {
    sm: 'w-4 h-4',
    md: 'w-6 h-6', 
    lg: 'w-8 h-8',
  }
  
  return (
    <Loader2 
      className={clsx(
        'animate-spin text-primary-600',
        sizes[size],
        className
      )} 
    />
  )
}
```

### Page Transitions
```tsx
// src/components/PageTransition.tsx
import { motion } from 'framer-motion'

const pageVariants = {
  initial: { opacity: 0, y: 20 },
  in: { opacity: 1, y: 0 },
  out: { opacity: 0, y: -20 },
}

const pageTransition = {
  type: 'tween',
  ease: 'anticipate',
  duration: 0.4,
}

export const PageTransition: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <motion.div
    initial="initial"
    animate="in"
    exit="out"
    variants={pageVariants}
    transition={pageTransition}
  >
    {children}
  </motion.div>
)
```

## 🚀 Build and Deployment

### Vite Configuration
```ts
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { TanStackRouterVite } from '@tanstack/router-vite-plugin'
import path from 'path'

export default defineConfig({
  plugins: [
    react(),
    TanStackRouterVite(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          router: ['@tanstack/react-router'],
          query: ['@tanstack/react-query'],
          ui: ['@headlessui/react', 'lucide-react'],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
```

### Environment Variables
#### Mobile-Responsive Booking Card
```tsx
// apps/rider/src/components/features/booking/BookingCard.tsx
import React from 'react'
import { format } from 'date-fns'
import { MapPin, Clock, Users, MoreVertical } from 'lucide-react'
import { Button } from '@luxsuv/ui'
import type { Booking } from '@luxsuv/api/types'

interface BookingCardProps {
  booking: Booking
  onView: () => void
  onEdit?: () => void
  onCancel?: () => void
}

export const BookingCard: React.FC<BookingCardProps> = ({
  booking,
  onView,
  onEdit,
  onCancel,
}) => {
  const statusConfig = {
    pending: { color: 'amber', label: 'Pending' },
    confirmed: { color: 'blue', label: 'Confirmed' },
    assigned: { color: 'purple', label: 'Driver Assigned' },
    on_trip: { color: 'green', label: 'In Progress' },
    completed: { color: 'gray', label: 'Completed' },
    canceled: { color: 'red', label: 'Canceled' },
  }

  const status = statusConfig[booking.status]

  return (
    <div className="bg-white rounded-2xl shadow-md border border-gray-100 overflow-hidden">
      {/* Status Header */}
      <div className={`h-2 bg-${status.color}-500`} />
      
      <div className="p-6">
        {/* Header */}
        <div className="flex justify-between items-start mb-4">
          <div>
            <p className="text-sm text-gray-500">Booking #{booking.id}</p>
            <span className={`inline-flex px-2 py-1 rounded-full text-xs font-medium bg-${status.color}-100 text-${status.color}-800`}>
              {status.label}
            </span>
          </div>
          <button className="p-1 hover:bg-gray-100 rounded">
            <MoreVertical className="w-4 h-4 text-gray-400" />
          </button>
        </div>

        {/* Route */}
        <div className="space-y-3 mb-4">
          <div className="flex items-start gap-3">
            <div className="w-3 h-3 rounded-full bg-green-500 mt-1" />
            <div>
              <p className="text-sm font-medium text-gray-900">{booking.pickup}</p>
              <p className="text-xs text-gray-500">Pickup location</p>
            </div>
          </div>
          
          <div className="flex items-start gap-3">
            <div className="w-3 h-3 rounded-full bg-red-500 mt-1" />
            <div>
              <p className="text-sm font-medium text-gray-900">{booking.dropoff}</p>
              <p className="text-xs text-gray-500">Destination</p>
            </div>
          </div>
        </div>

        {/* Details */}
        <div className="flex justify-between items-center text-sm text-gray-600 mb-4">
          <div className="flex items-center gap-1">
            <Clock className="w-4 h-4" />
            {format(new Date(booking.scheduled_at), 'MMM d, h:mm a')}
          </div>
          <div className="flex items-center gap-1">
            <Users className="w-4 h-4" />
            {booking.passengers} passenger{booking.passengers !== 1 ? 's' : ''}
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-2">
          <Button 
            onClick={onView}
            variant="secondary"
            size="sm"
            className="flex-1"
          >
            View Details
          </Button>
          {booking.status === 'pending' && onEdit && (
            <Button 
              onClick={onEdit}
              size="sm"
              className="flex-1"
            >
              Edit Booking
            </Button>
          )}
        </div>
      </div>
    </div>
  )

# Optional: Analytics and Monitoring
VITE_GA_MEASUREMENT_ID=
VITE_SENTRY_DSN=
VITE_HOTJAR_ID=

# Feature Flags
VITE_ENABLE_PWA=true
VITE_ENABLE_OFFLINE_MODE=false
```

### Production Build
```bash
# Build for production
npm run build

# Preview production build
npm run preview

# Type check
npm run type-check

# Run all tests
npm run test
npm run test:e2e
```

### Deployment (Vercel/Netlify)
```json
// vercel.json or _redirects for Netlify
{
  "rewrites": [
    { "source": "/(.*)", "destination": "/index.html" }
  ],
  "headers": [
    {
      "source": "/(.*).(js|css|woff2?|png|jpg|jpeg|gif|svg|ico)",
      "headers": [
        {
          "key": "Cache-Control",
          "value": "public, max-age=31536000, immutable"
        }
      ]
    }
  ]
}
```

## 📚 Development Workflow

### Scripts Setup
```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "test": "vitest",
    "test:e2e": "playwright test",
    "test:ui": "vitest --ui",
    "type-check": "tsc --noEmit",
    "lint": "eslint src --ext ts,tsx",
    "lint:fix": "eslint src --ext ts,tsx --fix",
    "format": "prettier --write src/**/*.{ts,tsx}",
    "storybook": "storybook dev -p 6006",
    "build-storybook": "storybook build"
  }
}
```

### Git Hooks (Husky)
```json
// package.json
{
  "husky": {
    "hooks": {
      "pre-commit": "lint-staged",
      "pre-push": "npm run type-check && npm run test"
    }
  },
  "lint-staged": {
    "*.{ts,tsx}": [
      "eslint --fix",
      "prettier --write"
    ]
  }
}
```

## 🔒 Security Best Practices

### Token Management
```tsx
// src/lib/auth/storage.ts
export const tokenStorage = {
  getAuthToken: (): string | null => {
    return localStorage.getItem('auth_token')
  },
  
  setAuthToken: (token: string): void => {
    localStorage.setItem('auth_token', token)
  },
  
  getRefreshToken: (): string | null => {
    return localStorage.getItem('refresh_token')
  },
  
  setRefreshToken: (token: string): void => {
    localStorage.setItem('refresh_token', token)
  },
  
  getGuestToken: (): string | null => {
    return localStorage.getItem('guest_session_token')
  },
  
  setGuestToken: (token: string): void => {
    localStorage.setItem('guest_session_token', token)
  },
  
  clearAll: (): void => {
    localStorage.removeItem('auth_token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('guest_session_token')
  },
}
```

### Input Sanitization
```tsx
// src/lib/utils/sanitization.ts
export const sanitizeInput = (input: string): string => {
  return input
    .trim()
    .replace(/[<>]/g, '') // Remove potential XSS characters
    .slice(0, 1000) // Limit length
}

export const sanitizeEmail = (email: string): string => {
  return email.toLowerCase().trim()
}

export const sanitizePhone = (phone: string): string => {
  return phone.replace(/[^\d+\-\s()]/g, '')
}
```

## 📊 Performance Optimization

### Code Splitting
```tsx
// src/routes/lazy-routes.ts
import { lazy } from 'react'

// Lazy load heavy components
export const AdminDashboard = lazy(() => import('../components/admin/AdminDashboard'))
export const BookingAnalytics = lazy(() => import('../components/analytics/BookingAnalytics'))
export const DriverMap = lazy(() => import('../components/map/DriverMap'))

// Usage in routes
import { Suspense } from 'react'
import { LoadingSpinner } from '../components/ui/LoadingSpinner'

function AdminRoute() {
  return (
    <Suspense fallback={<LoadingSpinner size="lg" />}>
      <AdminDashboard />
    </Suspense>
  )
}
// apps/admin/src/components/dashboard/LiveOperations.tsx
import React, { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { MapPin, Clock, AlertTriangle, CheckCircle } from 'lucide-react'
import { useWebSocket } from '../../hooks/useWebSocket'
import { adminAPI } from '@luxsuv/api'

export const LiveOperations: React.FC = () => {
  const { data: liveData, refetch } = useQuery({
    queryKey: ['live-operations'],
    queryFn: () => adminAPI.getLiveOperations(),
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  // WebSocket for real-time updates
  const { lastMessage } = useWebSocket('/v1/admin/live', {
    onMessage: (message) => {
      if (message.type === 'booking_update' || message.type === 'driver_update') {
        refetch()
      }
    },
  })

  return (
    <div className="space-y-6">
      {/* Status Overview */}
      <div className="grid grid-cols-4 gap-4">
        <StatusCard
          title="Active Trips"
          count={liveData?.activeTrips || 0}
          icon={<MapPin className="w-5 h-5" />}
          color="blue"
        />
        <StatusCard
          title="Pending Assignments"
          count={liveData?.pendingAssignments || 0}
          icon={<Clock className="w-5 h-5" />}
          color="amber"
        />
        <StatusCard
          title="Online Drivers"
          count={liveData?.onlineDrivers || 0}
          icon={<CheckCircle className="w-5 h-5" />}
          color="green"
        />
        <StatusCard
          title="Issues"
          count={liveData?.issues || 0}
          icon={<AlertTriangle className="w-5 h-5" />}
          color="red"
        />
      </div>

      {/* Live Activity Feed */}
      <div className="bg-white rounded-xl shadow-sm border">
        <div className="p-6 border-b">
          <h3 className="text-lg font-semibold">Live Activity</h3>
        </div>
        <div className="max-h-96 overflow-y-auto">
          {liveData?.recentActivity?.map((activity, index) => (
            <ActivityItem key={index} activity={activity} />
          ))}
        </div>
      </div>
    </div>
const StatusCard: React.FC<{
  title: string
  count: number
  icon: React.ReactNode
  color: string
}> = ({ title, count, icon, color }) => (
  <div className="bg-white p-4 rounded-lg border shadow-sm">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm text-gray-600">{title}</p>
        <p className="text-2xl font-bold text-gray-900">{count}</p>
      </div>
      <div className={`p-2 rounded-lg bg-${color}-100 text-${color}-600`}>
        {icon}
      </div>
    </div>
  </div>
)
```

### 📱 Driver App Features

#### Driver Status Toggle
```tsx
// apps/driver/src/components/status/AvailabilityToggle.tsx
import React from 'react'
import { Power, Zap } from 'lucide-react'
import { Button } from '@luxsuv/ui'
import { useDriverStatus } from '../../hooks/api/useDriverStatus'

export const AvailabilityToggle: React.FC = () => {
  const { status, toggleAvailability, isToggling } = useDriverStatus()

  return (
    <div className="bg-white rounded-2xl shadow-lg p-6 mx-4">
      <div className="text-center">
        <div className={clsx(
          'w-20 h-20 rounded-full mx-auto mb-4 flex items-center justify-center',
          status.available ? 'bg-green-100' : 'bg-gray-100'
        )}>
          {status.available ? (
            <Zap className="w-10 h-10 text-green-600" />
          ) : (
            <Power className="w-10 h-10 text-gray-400" />
          )}
        </div>
        
        <h2 className="text-2xl font-bold text-gray-900 mb-2">
          {status.available ? 'Online' : 'Offline'}
        </h2>
        
        <p className="text-gray-600 mb-6">
          {status.available 
            ? 'You\'re available to receive ride requests'
            : 'Go online to start receiving ride requests'
          }
        </p>
        
        <Button
          onClick={toggleAvailability}
          isLoading={isToggling}
```

## 🚀 Deployment Strategies

### Rider App Deployment (Vercel)
```json
// apps/rider/vercel.json
{
  "framework": "vite",
  "buildCommand": "cd ../.. && pnpm build --filter=rider",
  "outputDirectory": "dist",
  "installCommand": "cd ../.. && pnpm install",
  "env": {
    "VITE_API_URL": "https://api.luxsuv.com",
    "VITE_APP_TYPE": "rider"
  },
  "routes": [
    { "handle": "filesystem" },
    { "src": "/(.*)", "dest": "/index.html" }
  ]
}
```

### Admin Portal Deployment (Netlify)
```toml
# apps/admin/netlify.toml
[build]
  command = "cd ../.. && pnpm build --filter=admin"
  publish = "dist"
  
[build.environment]
  VITE_API_URL = "https://api.luxsuv.com"
  VITE_APP_TYPE = "admin"
  
[[redirects]]
  from = "/*"
  to = "/index.html"
  status = 200
  
[context.production.environment]
  VITE_ENVIRONMENT = "production"
  
[context.staging.environment]
  VITE_ENVIRONMENT = "staging"
  VITE_API_URL = "https://staging-api.luxsuv.com"
```

### Driver App PWA Configuration
```json
// apps/driver/public/manifest.json
{
  "name": "LuxSuv Driver",
  "short_name": "LuxSuv Driver",
  "description": "LuxSuv driver app for managing rides and assignments",
  "theme_color": "#10b981",
  "background_color": "#ffffff", 
  "display": "standalone",
  "orientation": "portrait",
  "scope": "/",
  "start_url": "/",
  "icons": [
    {
      "src": "icons/icon-192x192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "icons/icon-512x512.png", 
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any maskable"
    }
  ],
  "categories": ["business", "productivity"],
  "screenshots": [
    {
      "src": "screenshots/driver-home.png",
      "sizes": "540x720",
      "type": "image/png"
    }
  ]
}
```

## 📱 Mobile-First Driver App

### Touch-Optimized Assignment Interface
```tsx
// apps/driver/src/components/assignments/SwipeAssignment.tsx
import React, { useRef } from 'react'
import { motion, useMotionValue, useTransform, PanInfo } from 'framer-motion'
import { Check, X, Navigation } from 'lucide-react'
import type { Assignment } from '@luxsuv/api/types'

interface SwipeAssignmentProps {
  assignment: Assignment
  onAccept: () => void
  onDecline: () => void
              : 'bg-green-600 hover:bg-green-700'
          )}
      
      <img
        src={error ? fallback : src}
        alt={alt}
        onLoad={() => setIsLoading(false)}
        onError={() => {
export const SwipeAssignment: React.FC<SwipeAssignmentProps> = ({
  assignment,
  onAccept,
  onDecline,
}) => {
  const constraintsRef = useRef(null)
  const x = useMotionValue(0)
  const opacity = useTransform(x, [-150, 0, 150], [0.7, 1, 0.7])
  const acceptOpacity = useTransform(x, [50, 150], [0, 1])
  const declineOpacity = useTransform(x, [-150, -50], [1, 0])

  const handleDragEnd = (event: MouseEvent | TouchEvent | PointerEvent, info: PanInfo) => {
    if (info.offset.x > 150) {
      onAccept()
    } else if (info.offset.x < -150) {
      onDecline()
    }
  }

  return (
    <div ref={constraintsRef} className="relative bg-white rounded-2xl shadow-lg mx-4 overflow-hidden">
      {/* Background Actions */}
      <motion.div 
        className="absolute inset-0 bg-red-500 flex items-center justify-start pl-8"
        style={{ opacity: declineOpacity }}
      >
        <X className="w-8 h-8 text-white" />
      </motion.div>
      
      <motion.div 
        className="absolute inset-0 bg-green-500 flex items-center justify-end pr-8"
        style={{ opacity: acceptOpacity }}
      >
        <Check className="w-8 h-8 text-white" />
      </motion.div>

      {/* Draggable Card */}
      <motion.div
        drag="x"
        dragConstraints={constraintsRef}
        dragElastic={0.2}
        onDragEnd={handleDragEnd}
        style={{ x, opacity }}
        className="bg-white p-6 relative z-10"
      >
        {/* Assignment content similar to previous example */}
        <div className="space-y-4">
          {/* Trip details */}
          <div className="text-center">
            <p className="text-lg font-bold text-green-600">${assignment.estimated_fare}</p>
            <p className="text-sm text-gray-600">{assignment.trip_distance}km • {assignment.trip_duration}min</p>
          </div>
          
          {/* Swipe instruction */}
          <div className="text-center py-2">
            <p className="text-xs text-gray-500">
              Swipe left to decline • Swipe right to accept
            </p>
          </div>
        </div>
      </motion.div>
    </div>
  )
}
```

## 🌍 Internationalization (i18n)

## 🔄 State Management Examples

### Driver Status Store
```tsx
// src/lib/i18n/index.ts
interface Assignment {
  id: number
  pickup: string
  dropoff: string
  passenger_name: string
  scheduled_at: string
  estimated_fare: number
  expires_at: string
}

interface DriverStatusState {
  // Status
  isOnline: boolean
  isAvailable: boolean
  currentLocation: Location | null
  
  // Current assignment/trip
  currentAssignment: Assignment | null
  currentTrip: Trip | null
  
  // Pending assignments
  pendingAssignments: Assignment[]
  
  // Actions
  setOnline: (online: boolean) => void
  setAvailable: (available: boolean) => void
  updateLocation: (location: Location) => void
  setCurrentAssignment: (assignment: Assignment | null) => void
  addPendingAssignment: (assignment: Assignment) => void
  removePendingAssignment: (assignmentId: number) => void
  
  // Trip actions
  startTrip: (trip: Trip) => void
  completeTrip: () => void
}

export const useDriverStatusStore = create<DriverStatusState>()(
  subscribeWithSelector((set, get) => ({
    // Initial state
    isOnline: false,
    isAvailable: false,
    currentLocation: null,
    currentAssignment: null,
    currentTrip: null,
    pendingAssignments: [],
    
    // Actions
    setOnline: (online) => {
      set({ isOnline: online })
      if (!online) {
        set({ isAvailable: false, currentAssignment: null })
      }
    },
    
    setAvailable: (available) => {
      const { isOnline } = get()
      if (isOnline) {
        set({ isAvailable: available })
      }
    },
    
    updateLocation: (location) => {
      set({ currentLocation: location })
    },
    
    setCurrentAssignment: (assignment) => {
      set({ currentAssignment: assignment })
      if (assignment) {
        set({ isAvailable: false })
      }
    },
    
    addPendingAssignment: (assignment) => {
      set((state) => ({
        pendingAssignments: [...state.pendingAssignments, assignment],
      }))
    },
    
    removePendingAssignment: (assignmentId) => {
      set((state) => ({
        pendingAssignments: state.pendingAssignments.filter(a => a.id !== assignmentId),
      }))
    },
    
    startTrip: (trip) => {
      set({ currentTrip: trip, currentAssignment: null })
    },
    
    completeTrip: () => {
      set({ currentTrip: null, isAvailable: true })
    },
  }))
)

// Location tracking subscription
useDriverStatusStore.subscribe(
  (state) => state.isOnline,
  (isOnline) => {
    if (isOnline) {
      // Start location tracking
      startLocationTracking()
    } else {
      // Stop location tracking
      stopLocationTracking()
    }
  timestamp: number
)
```

## 🏃‍♂️ Getting Started Checklist

### Phase 1: Project Setup
- [ ] Create Vite React TypeScript project
- [ ] Install and configure dependencies
- [ ] Set up Tailwind CSS with design system
- [ ] Configure TypeScript paths and aliases
- [ ] Set up ESLint and Prettier
- [ ] Configure environment variables

### Phase 2: Core Infrastructure
- [ ] Implement API client with error handling
- [ ] Set up React Query with proper configuration
- [ ] Create Zustand stores for state management
- [ ] Implement TanStack Router file-based routing
- [ ] Set up error boundaries and fallbacks
- [ ] Configure toast notifications

### Phase 3: Design System
- [ ] Build base UI components (Button, Input, Modal, etc.)
- [ ] Implement responsive layout components
- [ ] Create loading states and skeletons
- [ ] Add animations and micro-interactions
- [ ] Implement dark mode support (optional)

### Phase 4: Authentication Features
- [ ] User registration and email verification
- [ ] User login with JWT handling
- [ ] Guest access flow with email codes
- [ ] Magic link authentication
- [ ] Session management and token refresh
- [ ] Route guards and protected routes

### Phase 5: Booking Features
- [ ] Guest booking creation form
- [ ] Authenticated user booking flow
- [ ] Booking list with filtering and pagination
- [ ] Individual booking view and management
- [ ] Booking editing with validation
- [ ] Booking cancellation with confirmation

### Phase 6: Advanced Features
- [ ] Real-time updates via WebSockets
- [ ] Offline support with service workers
- [ ] Push notifications for booking updates
- [ ] Advanced search and filtering
- [ ] Booking history and analytics
- [ ] Admin dashboard for management

### Phase 7: Testing and Quality
- [ ] Unit tests for components and hooks
- [ ] Integration tests for user flows
- [ ] E2E tests with Playwright
- [ ] Accessibility testing and WCAG compliance
- [ ] Performance testing and optimization
- [ ] Cross-browser testing

### Phase 8: Production Readiness
- [ ] Bundle optimization and code splitting
- [ ] SEO optimization with meta tags
- [ ] Analytics integration (Google Analytics)
- [ ] Error monitoring (Sentry)
- [ ] Performance monitoring (Web Vitals)
- [ ] CI/CD pipeline setup

## 📞 Support and Resources

### Useful Links
- [React Documentation](https://react.dev/)
- [TanStack Router Guide](https://tanstack.com/router)
- [TanStack Query Guide](https://tanstack.com/query)
- [Tailwind CSS Documentation](https://tailwindcss.com/)
- [Headless UI Components](https://headlessui.com/)
- [Zod Validation](https://zod.dev/)

### Development Tools
- **React DevTools** - Browser extension for React debugging
- **TanStack Query DevTools** - Built-in query state inspector
- **Redux DevTools** - For Zustand store debugging
- **Tailwind CSS IntelliSense** - VS Code extension
- **TypeScript Hero** - VS Code extension for imports

This comprehensive guide provides everything needed to implement a production-ready React frontend for the LuxSuv booking platform. Each section includes practical examples and follows modern React best practices.