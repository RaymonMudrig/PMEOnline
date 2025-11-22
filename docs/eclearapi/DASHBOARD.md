# eClear API Dashboard

## Overview
The eClear API now includes a web-based dashboard for viewing master data in a user-friendly interface.

## Features

### 📊 Three Tabbed Views

1. **Participants Tab**
   - View all registered participants
   - See borrow and lend eligibility status
   - Statistics: Total participants, borrow eligible, lend eligible

2. **Instruments Tab**
   - View all registered instruments
   - See eligibility status for lending/borrowing
   - Statistics: Total instruments, eligible, ineligible

3. **Accounts Tab**
   - View all registered accounts with limits
   - See Trade Limit and Pool Limit for each account
   - Linked to participants

### 🎨 Modern UI Features

- Gradient purple theme
- Responsive design (works on desktop and mobile)
- Real-time refresh buttons
- Status indicators (loading, success, error)
- Statistics cards for quick insights
- Hover effects and smooth animations
- Empty state handling with helpful messages

## Access

### URL
Once the eClear API is running, access the dashboard at:

```
http://localhost:8081/
http://localhost:8081/dashboard
```

### API Endpoints
The dashboard uses these REST endpoints:

```
GET /participant/list  - Returns all participants
GET /instrument/list   - Returns all instruments
GET /account/list      - Returns all accounts with limits
```

## Usage

1. **Start the eClear API service**:
   ```bash
   cd cmd/eclearapi
   go run main.go
   ```

2. **Load sample data** (optional):
   ```bash
   ./test.sh
   ```

3. **Open dashboard** in your browser:
   ```
   http://localhost:8081/
   ```

4. **Navigate tabs** by clicking on:
   - 👥 Participants
   - 📊 Instruments
   - 💼 Accounts

5. **Refresh data** by clicking the "🔄 Refresh" button on each tab

## Architecture

### Frontend
- Pure HTML/CSS/JavaScript (no frameworks)
- Fetch API for REST calls
- Modern CSS with gradients and animations
- Responsive grid layout

### Backend
- Query endpoints in `handlers/query.go`
- CORS middleware for browser access
- Embedded HTML template in main.go
- Real-time data from LedgerPoint

## Development

### Adding New Tabs
To add a new tab to the dashboard:

1. Add query handler method in `handlers/query.go`
2. Register endpoint in `main.go`
3. Update `dashboardHTML` constant with new tab HTML
4. Add JavaScript function to load data

### CORS Configuration
The dashboard includes CORS middleware to allow browser access:
- Allows all origins (`*`)
- Supports GET, POST, PUT, DELETE methods
- Allows Content-Type and Authorization headers

## Troubleshooting

### Dashboard shows "No data found"
- Run `./test.sh` to insert sample data
- Check that Kafka is running
- Verify LedgerPoint is ready (check logs)

### API endpoints return errors
- Check that eClear API is running on port 8081
- Verify Kafka connectivity
- Check browser console for network errors

### Data not refreshing
- Click the refresh button manually
- Check browser console for JavaScript errors
- Verify API endpoints are accessible

## Security Note

In production, you should:
- Restrict CORS to specific domains
- Add authentication/authorization
- Use HTTPS instead of HTTP
- Add rate limiting
