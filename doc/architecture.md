# Project Architecture & Technology Stack

## Overview
The project follows a full-stack architecture with:
- **Frontend**: Next.js-based application
- **Backend**: FastAPI service for API endpoints and business logic

## Technology Stack

### Frontend

#### Core Framework
- **Next.js 14.0.0**: Application framework providing React server components, routing, and API routes capability
- **React 18**: UI library with concurrent rendering features

#### Styling
- **Tailwind CSS 3.3.0**: Utility-first CSS framework
  - Custom color palette with `primary` (yellow/gold theme) and `mystic` (blue-gray) color schemes
  - Custom font families: Inter (sans-serif) and Playfair Display (serif)
  - Gradient utilities for background effects
- **PostCSS & Autoprefixer**: For CSS processing and vendor prefixing

#### Additional Libraries
- **Axios 1.6.0**: For API communication with backend services
- **Lucide React 0.309.0**: Icon library

### Backend

#### Core Framework
- **FastAPI 0.135.3**: Modern Python web framework for building API endpoints
- **uvicorn 0.42.0**: ASGI server for running the application

#### Data Handling
- **pydantic 2.12.5**: Data validation and settings management
- **peewee 3.10.0**: ORM for database interactions
- **agno 2.5.2**: Additional data processing library

#### Configuration
- **python-dotenv 1.2.2**: Environment variable management

## Project Structure

```
taro_agent/
├── frontend/
│   ├── app/
│   │   └── **/page.{js,ts,jsx,tsx}
│   ├── components/
│   ├── public/
│   ├── tailwind.config.js
│   ├── package.json
│   └── ... (Next.js standard structure)
├── backend/
│   ├── app/
│   │   ├── core/
│   │   │   └── config.py
│   │   ├── db/
│   │   │   ├── base.py
│   │   │   └── models.py
│   │   ├── api/
│   │   │   ├── endpoints/
│   │   │   │   ├── cards.py
│   │   │   │   └── readings.py
│   │   ├── agent/
│   │   │   └── interpreter.py
│   │   └── main.py
│   ├── run.py
│   └── ...
├── pyproject.toml
├── requirements.txt
└── doc/
    └── architecture.md (this document)
```

## Backend API Structure

- **Base Path**: `/api/v1`
- **Key Endpoints**:
  - `/api/v1/cards` - Card management
  - `/api/v1/readings` - Reading endpoints
- **Additional Routes**:
  - `/` - Root endpoint with project info
  - `/health` - Health check endpoint
  - `/docs` - Interactive API documentation (Swagger UI)

## Configuration Highlights

### Frontend (Tailwind)
- Content sources configured for `app/**/*` and `components/**/*` directories
- Custom theme extensions for colors, fonts, and gradients

### Backend (FastAPI)
- CORS configuration for secure cross-origin requests
- Database initialization in application lifespan
- Environment-based configuration via pydantic-settings

## Notes
- The project is a full-stack application with clear separation between frontend and backend
- Backend uses Peewee ORM with likely SQLite database (based on common FastAPI/Peewee usage)
- API endpoints are versioned under `/api/v1`
- Both frontend and backend use environment variables for configuration