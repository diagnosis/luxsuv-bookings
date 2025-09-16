# LuxSUV Bookings API

A luxury SUV booking service with guest access (passwordless) and registered user flows.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Authentication](#authentication)
- [API Endpoints](#api-endpoints)
- [Frontend Integration Guide](#frontend-integration-guide)
- [Error Handling](#error-handling)
- [Security Considerations](#security-considerations)
- [Testing](#testing)

## Architecture Overview

### Tech Stack
- **Backend**: Go with Chi router, PostgreSQL, JWT auth
- **Email**: SMTP (dev) / MailerSend (production)
- **Auth**: Passwordless guest access + traditional user accounts
- **API**: RESTful JSON API with structured error responses

### Key Concepts

**Guest Flow**: Passwordless booking system where users:
1. Create bookings without authentication
2. Request access via email (6-digit code + magic link)  
3. Get temporary guest JWT (30min) to manage their bookings
4. Can access bookings via session OR manage token (from email)

**Registered User Flow**: Traditional account-based booking (implemented but not focus)

## Authentication

### Guest Session Flow

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant API
    participant Email

    User->>Frontend: Create booking
    Frontend->>API: POST /v1/guest/bookings
    API-->>Frontend: {id, manage_token, status}
    
    User->>Frontend: Request access
    Frontend->>API: POST /v1/guest/access/request {email}
    API->>Email: Send 6-digit code + magic link
    API-->>Frontend: {message: "sent"}
    
    User->>Frontend: Enter 6-digit code
    Frontend->>API: POST /v1/guest/access/verify {email, code}
    API-->>Frontend: {session_token, expires_in}
    
    Note over Frontend: Store session_token (30min TTL)
    
    Frontend->>API: Subsequent requests with Bearer token
```

### Token Types

1. **Session Token** (JWT)
   - Role: `guest`
   - TTL: 30 minutes
   - Use: All authenticated guest operations
   - Header: `Authorization: Bearer <token>`

2. **Manage Token** (UUID)
   - Per-booking access token
   - No expiration
   - Use: Email deep-links, backup access
   - Query param: `?manage_token=<uuid>`

## API Endpoints

### Base URL
```
Development: http://localhost:8080
Production: https://your-domain.com
```

### Health Check
```http
GET /healthz
```
Response: `{"status":"ok"}`

---

### Guest Access (Passwordless Auth)

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

**Rate Limited**: 5 requests per minute per IP

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

**Response**: Same as verify code

---

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

**Validation Rules**:
- `scheduled_at`: Must be in the future
- `passengers`: 1-8 people
- `luggages`: 0-10 bags  
- `ride_type`: `"per_ride"` or `"hourly"`
- Email format validation
- Phone format validation

#### List My Bookings (Session Required)
```http
GET /v1/guest/bookings?limit=20&offset=0&status=pending
Authorization: Bearer <session_token>
```

**Response**: `200 OK`
```json
[
  {
    "id": 123,
    "status": "pending",
    "rider_name": "John Doe",
    "rider_email": "john@example.com",
    "rider_phone": "+1234567890",
    "pickup": "SFO Terminal 1",
    "dropoff": "Downtown Hotel",
    "scheduled_at": "2025-12-01T15:30:00Z",
    "notes": "2 large bags",
    "passengers": 2,
    "luggages": 2,
    "ride_type": "per_ride",
    "driver_id": null,
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T10:00:00Z"
  }
]
```

**Query Parameters**:
- `limit`: 1-100 (default: 20)
- `offset`: ≥0 (default: 0)
- `status`: `pending|confirmed|assigned|on_trip|completed|canceled`

#### Get Single Booking

**With Session**:
```http
GET /v1/guest/bookings/123
Authorization: Bearer <session_token>
```

**With Manage Token** (no auth needed):
```http
GET /v1/guest/bookings/123?manage_token=550e8400-e29b-41d4-a716-446655440000
```

**Response**: `200 OK` - Full booking object (same as create response plus all fields)

#### Update Booking

**With Session**:
```http
PATCH /v1/guest/bookings/123
Authorization: Bearer <session_token>
Content-Type: application/json

{
  "notes": "Updated notes",
  "passengers": 3,
  "pickup": "SFO Terminal 2"
}
```

**With Manage Token**:
```http
PATCH /v1/guest/bookings/123?manage_token=550e8400-e29b-41d4-a716-446655440000
Content-Type: application/json

{
  "scheduled_at": "2025-12-01T16:00:00Z"
}
```

**Patchable Fields**:
- `rider_name`, `rider_phone`
- `pickup`, `dropoff`, `notes`
- `scheduled_at` (must be future)
- `passengers` (1-8), `luggages` (0-10)
- `ride_type`

#### Cancel Booking

**With Session**:
```http
DELETE /v1/guest/bookings/123
Authorization: Bearer <session_token>
```

**With Manage Token**:
```http
DELETE /v1/guest/bookings/123?manage_token=550e8400-e29b-41d4-a716-446655440000
```

**Response**: `204 No Content`

## Frontend Integration Guide

### Recommended Flow

#### 1. Guest Booking Creation
```javascript
// Create booking (no auth needed)
const createBooking = async (bookingData) => {
  const response = await fetch('/v1/guest/bookings', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      // Optional: add idempotency for duplicate prevention
      'Idempotency-Key': `booking-${Date.now()}-${Math.random()}`
    },
    body: JSON.stringify(bookingData)
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to create booking');
  }

  return await response.json();
};
```

#### 2. Request Access & Verification
```javascript
// Request access code
const requestAccess = async (email) => {
  const response = await fetch('/v1/guest/access/request', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email })
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to send access code');
  }

  return await response.json();
};

// Verify code and get session
const verifyCode = async (email, code) => {
  const response = await fetch('/v1/guest/access/verify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, code })
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Invalid code');
  }

  const { session_token, expires_in } = await response.json();
  
  // Store session token
  localStorage.setItem('guest_session', JSON.stringify({
    token: session_token,
    expires: Date.now() + (expires_in * 1000),
    email
  }));

  return session_token;
};
```

#### 3. Session Management
```javascript
class GuestSession {
  static getToken() {
    const stored = localStorage.getItem('guest_session');
    if (!stored) return null;

    const session = JSON.parse(stored);
    if (Date.now() > session.expires) {
      localStorage.removeItem('guest_session');
      return null;
    }

    return session.token;
  }

  static isAuthenticated() {
    return this.getToken() !== null;
  }

  static getEmail() {
    const stored = localStorage.getItem('guest_session');
    return stored ? JSON.parse(stored).email : null;
  }

  static logout() {
    localStorage.removeItem('guest_session');
  }
}
```

#### 4. API Client
```javascript
class BookingAPI {
  static async request(endpoint, options = {}) {
    const token = GuestSession.getToken();
    
    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { 'Authorization': `Bearer ${token}` }),
        ...options.headers
      },
      ...options
    };

    const response = await fetch(`/v1${endpoint}`, config);
    
    if (!response.ok) {
      const error = await response.json();
      if (response.status === 401) {
        GuestSession.logout();
        // Trigger re-authentication flow
        window.location.href = '/guest-access';
        return;
      }
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.status === 204 ? null : await response.json();
  }

  static listBookings(params = {}) {
    const query = new URLSearchParams(params).toString();
    return this.request(`/guest/bookings${query ? '?' + query : ''}`);
  }

  static getBooking(id, manageToken = null) {
    const query = manageToken ? `?manage_token=${manageToken}` : '';
    return this.request(`/guest/bookings/${id}${query}`);
  }

  static updateBooking(id, updates, manageToken = null) {
    const query = manageToken ? `?manage_token=${manageToken}` : '';
    return this.request(`/guest/bookings/${id}${query}`, {
      method: 'PATCH',
      body: JSON.stringify(updates)
    });
  }

  static cancelBooking(id, manageToken = null) {
    const query = manageToken ? `?manage_token=${manageToken}` : '';
    return this.request(`/guest/bookings/${id}${query}`, {
      method: 'DELETE'
    });
  }
}
```

### URL Patterns

#### Guest Dashboard
```
/guest/bookings → List all bookings (requires session)
/guest/bookings/123 → View booking (session or from manage link)
/guest/bookings/123/edit → Edit booking
```

#### Deep Links (Email)
```
/guest/bookings/123?manage_token=xxx → Direct booking access
/guest/access?token=xxx → Magic link authentication
```

### State Management (React Example)

```javascript
// BookingContext.jsx
import React, { createContext, useContext, useReducer, useEffect } from 'react';

const BookingContext = createContext();

const initialState = {
  user: {
    isAuthenticated: false,
    email: null,
    sessionToken: null
  },
  bookings: [],
  loading: false,
  error: null
};

function bookingReducer(state, action) {
  switch (action.type) {
    case 'SET_LOADING':
      return { ...state, loading: action.payload };
    
    case 'SET_ERROR':
      return { ...state, error: action.payload, loading: false };
    
    case 'SET_USER':
      return { 
        ...state, 
        user: { 
          isAuthenticated: true, 
          ...action.payload 
        }, 
        error: null 
      };
    
    case 'LOGOUT':
      GuestSession.logout();
      return { 
        ...state, 
        user: { isAuthenticated: false, email: null, sessionToken: null },
        bookings: []
      };
    
    case 'SET_BOOKINGS':
      return { ...state, bookings: action.payload, loading: false };
    
    case 'ADD_BOOKING':
      return { ...state, bookings: [action.payload, ...state.bookings] };
    
    case 'UPDATE_BOOKING':
      return {
        ...state,
        bookings: state.bookings.map(b => 
          b.id === action.payload.id ? action.payload : b
        )
      };
    
    case 'REMOVE_BOOKING':
      return {
        ...state,
        bookings: state.bookings.filter(b => b.id !== action.payload)
      };
    
    default:
      return state;
  }
}

export function BookingProvider({ children }) {
  const [state, dispatch] = useReducer(bookingReducer, initialState);

  // Initialize from localStorage
  useEffect(() => {
    const token = GuestSession.getToken();
    const email = GuestSession.getEmail();
    
    if (token && email) {
      dispatch({
        type: 'SET_USER',
        payload: { email, sessionToken: token }
      });
    }
  }, []);

  // Load bookings when authenticated
  useEffect(() => {
    if (state.user.isAuthenticated) {
      loadBookings();
    }
  }, [state.user.isAuthenticated]);

  const loadBookings = async () => {
    dispatch({ type: 'SET_LOADING', payload: true });
    try {
      const bookings = await BookingAPI.listBookings();
      dispatch({ type: 'SET_BOOKINGS', payload: bookings });
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error.message });
    }
  };

  const login = async (email, code) => {
    try {
      const sessionToken = await verifyCode(email, code);
      dispatch({
        type: 'SET_USER',
        payload: { email, sessionToken }
      });
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error.message });
      throw error;
    }
  };

  const logout = () => {
    dispatch({ type: 'LOGOUT' });
  };

  const value = {
    ...state,
    actions: {
      login,
      logout,
      loadBookings,
      // ... other actions
    }
  };

  return (
    <BookingContext.Provider value={value}>
      {children}
    </BookingContext.Provider>
  );
}

export const useBooking = () => {
  const context = useContext(BookingContext);
  if (!context) {
    throw new Error('useBooking must be used within BookingProvider');
  }
  return context;
};
```

## Error Handling

### Error Response Format
```json
{
  "error": "Human readable error message",
  "code": "ERROR_CODE",
  "details": "Additional context (optional)"
}
```

### Common Error Codes

| Code | HTTP | Description | Action |
|------|------|-------------|---------|
| `INVALID_INPUT` | 400 | Bad request data | Show field errors |
| `UNAUTHORIZED` | 401 | Missing/invalid auth | Redirect to login |
| `FORBIDDEN` | 403 | Access denied | Show permission error |
| `NOT_FOUND` | 404 | Resource not found | Show 404 page |
| `CONFLICT` | 409 | Business logic conflict | Show specific message |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests | Show retry timer |
| `EXPIRED_TOKEN` | 401 | Token expired | Refresh or re-auth |
| `PAST_DATETIME` | 400 | Date in past | Highlight field |

### Frontend Error Handling
```javascript
const handleAPIError = (error, context) => {
  switch (error.code) {
    case 'RATE_LIMIT_EXCEEDED':
      showToast('Too many requests. Please wait before trying again.', 'warning');
      break;
    
    case 'EXPIRED_TOKEN':
      logout();
      showToast('Your session expired. Please verify your email again.', 'info');
      break;
    
    case 'PAST_DATETIME':
      setFieldError('scheduled_at', 'Please select a future date and time');
      break;
    
    case 'INVALID_INPUT':
      if (context.field) {
        setFieldError(context.field, error.error);
      } else {
        showFormError(error.error);
      }
      break;
    
    default:
      showToast(error.error || 'Something went wrong. Please try again.', 'error');
  }
};
```

## Security Considerations

### Frontend Security

1. **Token Storage**
   - Use `localStorage` for session tokens (30min TTL)
   - Never store `manage_token` in localStorage (comes from URL)
   - Clear tokens on logout/expiry

2. **Request Validation**
   - Validate inputs client-side before sending
   - Always handle server validation errors
   - Sanitize display data

3. **Deep Link Handling**
   - Validate `manage_token` format (UUID)
   - Don't persist manage tokens across sessions
   - Use session auth when available

4. **Rate Limiting**
   - Show loading states to prevent button mashing
   - Implement client-side cooldowns for access requests
   - Handle 429 responses gracefully

### CORS Configuration

The API allows these origins:
- `http://localhost:5173` (Vite dev)
- `http://localhost:3000` (Create React App)

Headers allowed:
- `Authorization` (Bearer tokens)
- `Content-Type` (JSON)
- `Idempotency-Key` (duplicate prevention)

## Testing

### Unit Tests (Jest Example)
```javascript
// BookingAPI.test.js
import { BookingAPI, GuestSession } from '../src/api';

describe('BookingAPI', () => {
  beforeEach(() => {
    fetchMock.resetMocks();
    localStorage.clear();
  });

  test('lists bookings with auth token', async () => {
    // Mock session
    localStorage.setItem('guest_session', JSON.stringify({
      token: 'mock-token',
      expires: Date.now() + 1800000,
      email: 'test@example.com'
    }));

    fetchMock.mockResponseOnce(JSON.stringify([
      { id: 1, status: 'pending', rider_name: 'Test' }
    ]));

    const bookings = await BookingAPI.listBookings();

    expect(fetch).toHaveBeenCalledWith('/v1/guest/bookings', {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer mock-token'
      }
    });
    expect(bookings).toHaveLength(1);
  });

  test('handles 401 by logging out', async () => {
    localStorage.setItem('guest_session', JSON.stringify({
      token: 'expired-token',
      expires: Date.now() + 1800000,
      email: 'test@example.com'
    }));

    fetchMock.mockResponseOnce(
      JSON.stringify({ error: 'Token expired' }), 
      { status: 401 }
    );

    // Mock window.location.href
    delete window.location;
    window.location = { href: '' };

    await expect(BookingAPI.listBookings()).resolves.toBeUndefined();
    expect(localStorage.getItem('guest_session')).toBeNull();
    expect(window.location.href).toBe('/guest-access');
  });
});
```

### Integration Tests
```javascript
// BookingFlow.test.js
import { render, screen, waitFor, userEvent } from '@testing-library/react';
import BookingFlow from '../src/components/BookingFlow';

describe('Guest Booking Flow', () => {
  test('complete booking and access flow', async () => {
    render(<BookingFlow />);

    // Create booking
    await userEvent.type(screen.getByLabelText(/name/i), 'John Doe');
    await userEvent.type(screen.getByLabelText(/email/i), 'john@example.com');
    // ... fill other fields

    await userEvent.click(screen.getByRole('button', { name: /book now/i }));

    await waitFor(() => {
      expect(screen.getByText(/check your email/i)).toBeInTheDocument();
    });

    // Verify code
    await userEvent.type(screen.getByLabelText(/code/i), '123456');
    await userEvent.click(screen.getByRole('button', { name: /verify/i }));

    await waitFor(() => {
      expect(screen.getByText(/booking confirmed/i)).toBeInTheDocument();
    });
  });
});
```

---

## Development Setup

### Prerequisites
- Node.js 18+
- Go 1.21+
- PostgreSQL 15+
- SMTP server (Mailpit for dev)

### Environment Variables
```bash
# Backend (.env)
PORT=8080
DATABASE_URL=postgres://user:pass@localhost:5432/luxsuv
JWT_SECRET=your-secret-key

# SMTP (Development)
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_FROM=noreply@luxsuv.local

# MailerSend (Production)
MAILERSEND_API_KEY=your-api-key
MAILER_FROM=noreply@yourdomain.com
```

### Frontend (.env)
```bash
REACT_APP_API_URL=http://localhost:8080
REACT_APP_ENVIRONMENT=development
```

### Quick Start
```bash
# Backend
make db/up        # Start PostgreSQL
make migrate/up   # Run migrations  
make run          # Start API server

# Frontend  
npm install
npm start         # Start dev server
```

---

## API Rate Limits

| Endpoint | Limit | Window | Scope |
|----------|-------|---------|-------|
| `/v1/guest/access/request` | 5 requests | 1 minute | Per IP |
| All other endpoints | No limit | - | - |

---

## Support

For questions or issues:
- API Documentation: This README
- Backend Issues: Check server logs
- Frontend Integration: Follow the patterns in this guide

The API is designed to be frontend-agnostic - use any framework (React, Vue, Svelte, vanilla JS) following these patterns.