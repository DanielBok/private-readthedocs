Private Read the Docs
=====================

A place to house all the sphinx documents. 

## Getting Started

To get started register an account using Postman or curl. Most of the API
requires Basic Auth to authenticate and execute. 

## API

### `/api/account/` [GET]

Validates the account from the BasicAuth headers.

### `/api/account/` [POST]

Creates an account. Any packages that are uploaded belong to this account
and can only be updated or removed by the account. 

```typescript
type Request = {
    username: string;
    password: string;
}
```

### `/api/account/` [PUT]

Updates the account. The user is authenticated with Basic Auth. The payload
will update the specified account via the **id** if the user is authorized to 
do so.

```typescript
type Request = {
    id: number;
    username: string;
    password: string;
}
```

### `/api/account/{username}` [DELETE]

Removes the account specified by `username`. Only admins or the account owner
(specified by the BasicAuth header) can execute request.

### `/api/project/` [GET]

Get all projects.

### `/api/project/{username}` [GET]

Get all projects from specified user.

### `/api/project/` [POST]

Uploads a new project. If it exists, replaces existing. User must have valid
credentials (must already own project) to do so.

### `/api/project/{title}` [DELETE]

Removes project. Caller must be owner of project.
