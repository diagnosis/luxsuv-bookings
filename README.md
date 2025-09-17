# LuxSuv Booking System

A modern luxury SUV booking platform with a Go backend API and React frontend featuring guest bookings, user authentication, and real-time booking management.

## 🚀 Tech Stack

### Backend
- **Go** with Chi router
- **PostgreSQL** database
- **JWT** authentication
- **Email verification** (SMTP/MailerSend)
- **Docker** for database

### Frontend
- **React 18** with TypeScript
- **Vite** for build tooling
- **TailwindCSS** for styling
- **TanStack Router** for file-based routing
- **TanStack Query** for server state management
- **React Hook Form** for form handling

## 📁 Project Structure

```
├── cmd/api/              # Go API entry point
├── internal/
│   ├── database/         # Database connection
│   ├── domain/           # Business models
│   ├── http/
│   │   ├── handlers/     # API route handlers
│   │   └── middleware/   # HTTP middleware
│   ├── platform/
│   │   ├── auth/         # JWT authentication
│   │   └── mailer/       # Email services
│   └── repo/             # Database repositories
├── migrations/           # Database migrations
├── frontend/             # React frontend (to be created)
│   ├── src/
│   │   ├── routes/       # File-based routing
│   │   ├── components/   # React components
│   │   ├── hooks/        # Custom hooks
│   │   ├── lib/          # Utilities and API client
│   │   └── types/        # TypeScript definitions
│   └── public/
└── docker-compose.yml    # Database setup
```

## 🚦 Getting Started

### Prerequisites
- Go 1.24+
- Node.js 18+
- Docker and Docker Compose
- Make (optional)

### Backend Setup

1. **Start the database**
```bash
make db/up
# or
docker compose up -d db
```

2. **Run migrations**
```bash
make migrate/up
# or manually:
# . ./.env && goose -dir ./migrations postgres "$DATABASE_URL" up
```

3. **Set environment variables**
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. **Start the API server**
```bash
make run
# or
go run ./cmd/api
```

The API will be available at `http://localhost:8080`

### Frontend Setup

1. **Create and setup the frontend**
```bash
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install

# Install TanStack Router and Query
npm install @tanstack/react-router @tanstack/react-query @tanstack/router-devtools @tanstack/router-vite-plugin

# Install TailwindCSS
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p

# Install additional dependencies
npm install react-hook-form @hookform/resolvers zod axios date-fns lucide-react
```

2. **Start the development server**
```bash
cd frontend
npm run dev
```

The frontend will be available at `http://localhost:5173`

## 🔌 API Endpoints

### Guest Bookings
```
POST   /v1/guest/bookings           # Create booking
GET    /v1/guest/bookings           # List bookings (requires session)
GET    /v1/guest/bookings/{id}      # Get booking (by manage_token or session)
PATCH  /v1/guest/bookings/{id}      # Update booking
DELETE /v1/guest/bookings/{id}      # Cancel booking
```

### Guest Access
```
POST   /v1/guest/access/request     # Request access code
POST   /v1/guest/access/verify      # Verify code -> get session
POST   /v1/guest/access/magic       # Magic link login
```

### User Authentication
```
POST   /v1/auth/register            # User registration
POST   /v1/auth/login               # User login
POST   /v1/auth/verify-email        # Email verification
```

### Authenticated User Bookings
```
GET    /v1/rider/bookings           # List user bookings
GET    /v1/rider/bookings/{id}      # Get user booking
POST   /v1/rider/bookings           # Create user booking
DELETE /v1/rider/bookings/{id}      # Cancel user booking
```

## 🎯 Key Features

### Backend Features
- **Guest Bookings**: Anonymous users can create bookings with manage tokens
- **Email Access**: Guest users get 6-digit codes + magic links via email
- **User Registration**: Full registration with email verification
- **JWT Authentication**: Secure token-based authentication
- **Booking Management**: Full CRUD operations for bookings
- **Status Tracking**: Booking lifecycle management
- **Email Integration**: SMTP and MailerSend support

### Frontend Features (Planned)
- **File-based Routing**: Automatic route generation with TanStack Router
- **Server State**: Optimistic updates with TanStack Query
- **Form Management**: Type-safe forms with React Hook Form + Zod
- **Responsive Design**: Mobile-first with Tailwind CSS
- **Guest Flow**: Seamless booking without registration
- **User Dashboard**: Authenticated user booking management
- **Real-time Updates**: Live booking status updates

## 🗄️ Database Schema

### Main Tables
- `bookings` - Ride bookings with guest/user association
- `users` - Registered user accounts
- `guest_access_codes` - Temporary access codes for guests
- `email_verification_tokens` - Email verification system
- `booking_access_tokens` - One-time booking access tokens

### Booking Status Flow
```
pending → confirmed → assigned → on_trip → completed
                            ↘ canceled
```

## 🔧 Development

### Backend Commands
```bash
make run              # Start API server
make db/up            # Start database
make db/down          # Stop database
make db/psql          # Connect to database
make migrate/up       # Run migrations
make migrate/down     # Rollback migrations
```

### Frontend Commands
```bash
cd frontend
npm run dev           # Start dev server
npm run build         # Build for production
npm run preview       # Preview production build
npm run lint          # Run ESLint
npm run type-check    # Run TypeScript check
```

### Testing API
Use the provided `test.http` file with your HTTP client to test all endpoints:

```bash
# Example guest booking flow
POST /v1/guest/bookings     # Create booking
POST /v1/guest/access/request # Request access
POST /v1/guest/access/verify  # Verify code
GET  /v1/guest/bookings     # List with session
```

## 🚀 Deployment

### Backend
1. Build the Go binary
2. Set production environment variables
3. Run database migrations
4. Deploy with your preferred method (Docker, systemd, etc.)

### Frontend
1. Build the React application: `npm run build`
2. Serve the `dist` folder with your preferred static hosting
3. Configure API base URL for production

## 📝 Environment Variables

### Backend (.env)
```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/luxsuv-co?sslmode=disable
JWT_SECRET=your-production-secret
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_FROM=dev@luxsuv.local
MAILERSEND_API_KEY=your-mailersend-key
MAILER_FROM=noreply@luxsuv.com
PORT=8080
```

### Frontend (.env.local)
```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_APP_NAME=LuxSuv Bookings
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.