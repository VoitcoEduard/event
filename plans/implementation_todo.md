# DataView Feature Implementation Summary

This task involved implementing a new "DataView" feature for the Admin Event page, allowing users to view module-generated data in a spreadsheet-like format using Handsontable. The implementation required changes to both the backend Go API and the frontend Vue.js application.

## Backend Changes

1.  **API Endpoint**: Added a new `GET /events/{eventId}/modules/{moduleId}/responses` endpoint in `backend/handlers/events.go` to fetch responses for a specific module from the MongoDB `responses` collection.
2.  **Debug Logging**: Refactored the `DebugLog` functionality by moving `logger.go` and `logger_debug.go` into a new `backend/utils` package (`log.go` and `log_debug.go` respectively). This allowed `DebugLog` to be imported and used consistently across `main.go` and `handlers/events.go`, adhering to Go's build tag conventions.
3.  **Route Registration**: Registered the new API endpoint in `backend/main.go` within the existing event routing logic.

## Frontend Changes

1.  **Handsontable Integration**: Added `@handsontable/vue3` and `handsontable` dependencies to `frontend/package.json` and installed them.
2.  **DataView Component**: Created a new `frontend/src/components/DataView.vue` component. This component:
    *   Accepts `eventId` and `moduleId` as props.
    *   Fetches responses from the new backend API endpoint.
    *   Transforms the fetched JSON data into a tabular format suitable for Handsontable.
    *   Displays the data in a read-only Handsontable spreadsheet.
3.  **Admin Event Details Page Update**: Modified `frontend/src/views/AdminEventDetailsScreen.vue` to:
    *   Import and utilize the new `DataView.vue` component.
    *   Introduce state variables (`showDataViewModal`, `selectedDataModule`) to control the visibility and data of the DataView modal.
    *   Add a "Data" button to each module in the event's module list.
    *   Implement a `showData` method to open the DataView modal when the button is clicked, passing the relevant module information.
    *   Integrate a modal overlay to display the `DataView` component, ensuring `eventId` and `moduleId` are passed correctly.
    *   Added necessary CSS styling for the new button and modal elements.

## How to Run the Application

To run the application and test the new DataView feature:

### 1. Start the Backend

Navigate to the `backend` directory and run the compiled server:

```bash
cd backend
./server
```

*(Note: Ensure your MongoDB instance is running and accessible to the backend.)*

### 2. Start the Frontend

Navigate to the `frontend` directory and start the development server:

```bash
cd frontend
npm run dev
```

### 3. Access the Application

Open your web browser and go to `http://localhost:5173/` (or the address shown by `npm run dev`).

**To test the DataView feature:**

1.  Log in as an administrator (if applicable) or navigate to an event administration page.
2.  Create a new event and add a module (e.g., a "Form" module).
3.  Submit some responses to this module (e.g., from an incognito window or another browser tab by connecting to the event's PIN).
4.  On the Admin Event Details screen, locate the module you added and click the new "Data" button.
5.  A modal should appear displaying the submitted responses in a Handsontable spreadsheet.