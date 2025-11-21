# Waiting Room - Postman Testing Guide

Hướng dẫn test luồng Waiting Room với Postman.

## Prerequisites

1. Server đang chạy: `make run`
2. Database đã migrate
3. Có 2 user accounts (1 host, 1 participant)
4. Postman đã cài đặt

## Setup Environment Variables trong Postman

Tạo environment với các biến:

```
BASE_URL: http://localhost:8080/api/v1
HOST_TOKEN: <token của host sau khi login>
PARTICIPANT_TOKEN: <token của participant sau khi login>
ROOM_ID: <sẽ được set tự động>
PARTICIPANT_ID: <sẽ được set tự động>
```

---

## Test Flow: Waiting Room End-to-End

### **STEP 1: Login as Host** 

**Request:** `POST {{BASE_URL}}/auth/login`

**Headers:**
```json
Content-Type: application/json
```

**Body:**
```json
{
  "email": "host@example.com",
  "password": "password123"
}
```

**Response:** 
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "user": {
    "id": "host-uuid",
    "email": "host@example.com",
    "name": "Host User"
  }
}
```

**Action:** 
- Copy `access_token` 
- Set vào Postman Environment → `HOST_TOKEN`

---

### **STEP 2: Login as Participant**

**Request:** `POST {{BASE_URL}}/auth/login`

**Headers:**
```json
Content-Type: application/json
```

**Body:**
```json
{
  "email": "participant@example.com",
  "password": "password123"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "...",
  "user": {
    "id": "participant-uuid",
    "email": "participant@example.com",
    "name": "Participant User"
  }
}
```

**Action:** 
- Copy `access_token`
- Set vào Postman Environment → `PARTICIPANT_TOKEN`

---

### **STEP 3: Host Creates Room**

**Request:** `POST {{BASE_URL}}/rooms`

**Headers:**
```json
Content-Type: application/json
Authorization: Bearer {{HOST_TOKEN}}
```

**Body:**
```json
{
  "name": "Test Waiting Room",
  "description": "Testing waiting room functionality",
  "type": "public",
  "max_participants": 10
}
```

**Response:**
```json
{
  "room": {
    "id": "room-uuid-123",
    "name": "Test Waiting Room",
    "status": "scheduled",
    "host_id": "host-uuid",
    "max_participants": 10,
    "current_participants": 0
  },
  "livekit_token": "...",
  "livekit_url": "ws://localhost:7880"
}
```

**Action:**
- Copy `room.id`
- Set vào Postman Environment → `ROOM_ID`

---

### **STEP 4: Participant Joins Room (Enters Waiting Room)**

> **Note:** Hiện tại API chưa có logic tự động đưa vào waiting room khi join. 
> Bạn cần update `JoinRoom` handler để set status='waiting' cho non-host users.

**Request:** `POST {{BASE_URL}}/rooms/{{ROOM_ID}}/join`

**Headers:**
```json
Content-Type: application/json
Authorization: Bearer {{PARTICIPANT_TOKEN}}
```

**Expected Behavior:**
- Non-host users sẽ được tạo participant với `status='waiting'`
- Response trả về thông báo "You are in the waiting room"

**Expected Response:**
```json
{
  "message": "You are in the waiting room. Waiting for host approval.",
  "participant": {
    "id": "participant-record-uuid",
    "room_id": "room-uuid-123",
    "user_id": "participant-uuid",
    "status": "waiting",
    "role": "participant"
  }
}
```

**Action:**
- Copy `participant.id`
- Set vào Postman Environment → `PARTICIPANT_ID`

---

### **STEP 5: Host Views Waiting List**

**Request:** `GET {{BASE_URL}}/rooms/{{ROOM_ID}}/participants/waiting`

**Headers:**
```json
Authorization: Bearer {{HOST_TOKEN}}
```

**Expected Response:**
```json
{
  "participants": [
    {
      "id": "participant-record-uuid",
      "room_id": "room-uuid-123",
      "user_id": "participant-uuid",
      "user": {
        "id": "participant-uuid",
        "name": "Participant User",
        "email": "participant@example.com",
        "avatar_url": "..."
      },
      "status": "waiting",
      "role": "participant",
      "created_at": "2025-11-20T10:00:00Z"
    }
  ],
  "total": 1
}
```

**Validation:**
- ✅ Status code: 200
- ✅ Array contains waiting participants
- ✅ Each participant has status='waiting'

---

### **STEP 6A: Host Admits Participant** ✅

**Request:** `POST {{BASE_URL}}/rooms/{{ROOM_ID}}/participants/{{PARTICIPANT_ID}}/admit`

**Headers:**
```json
Authorization: Bearer {{HOST_TOKEN}}
```

**Expected Response:**
```json
{
  "message": "participant admitted successfully"
}
```

**Validation:**
- ✅ Status code: 200
- ✅ Participant status changed to 'joined'
- ✅ Room's current_participants incremented

**Verify:** GET `/rooms/{{ROOM_ID}}/participants` should show participant with `status='joined'`

---

### **STEP 6B: Host Denies Participant** ❌ (Alternative to 6A)

**Request:** `POST {{BASE_URL}}/rooms/{{ROOM_ID}}/participants/{{PARTICIPANT_ID}}/deny`

**Headers:**
```json
Content-Type: application/json
Authorization: Bearer {{HOST_TOKEN}}
```

**Body (Optional):**
```json
{
  "reason": "Room is full" 
}
```

**Expected Response:**
```json
{
  "message": "participant denied"
}
```

**Validation:**
- ✅ Status code: 200
- ✅ Participant status changed to 'denied'
- ✅ Room's current_participants NOT incremented

---

### **STEP 7: Verify Participant Status**

**Request:** `GET {{BASE_URL}}/rooms/{{ROOM_ID}}/participants`

**Headers:**
```json
Authorization: Bearer {{HOST_TOKEN}}
```

**Expected Response (if admitted):**
```json
{
  "participants": [
    {
      "id": "host-participant-uuid",
      "status": "joined",
      "role": "host"
    },
    {
      "id": "participant-record-uuid",
      "status": "joined",
      "role": "participant"
    }
  ],
  "total": 2
}
```

**Expected Response (if denied):**
```json
{
  "participants": [
    {
      "id": "host-participant-uuid",
      "status": "joined",
      "role": "host"
    },
    {
      "id": "participant-record-uuid",
      "status": "denied",
      "role": "participant"
    }
  ],
  "total": 2
}
```

---

## Error Test Cases

### **Test 1: Non-Host Cannot View Waiting List**

**Request:** `GET {{BASE_URL}}/rooms/{{ROOM_ID}}/participants/waiting`

**Headers:**
```json
Authorization: Bearer {{PARTICIPANT_TOKEN}}
```

**Expected Response:**
```json
{
  "error": "not_host",
  "message": "user is not the host"
}
```

**Expected Status:** `403 Forbidden`

---

### **Test 2: Non-Host Cannot Admit Participants**

**Request:** `POST {{BASE_URL}}/rooms/{{ROOM_ID}}/participants/{{PARTICIPANT_ID}}/admit`

**Headers:**
```json
Authorization: Bearer {{PARTICIPANT_TOKEN}}
```

**Expected Response:**
```json
{
  "error": "not_host",
  "message": "user is not the host"
}
```

**Expected Status:** `403 Forbidden`

---

### **Test 3: Cannot Admit Already Joined Participant**

**Setup:** Participant đã được admit (status='joined')

**Request:** `POST {{BASE_URL}}/rooms/{{ROOM_ID}}/participants/{{PARTICIPANT_ID}}/admit`

**Headers:**
```json
Authorization: Bearer {{HOST_TOKEN}}
```

**Expected Response:**
```json
{
  "error": "invalid_participant_status",
  "message": "invalid participant status for this operation"
}
```

**Expected Status:** `400 Bad Request` or `409 Conflict`

---

### **Test 4: Cannot Admit to Full Room**

**Setup:** 
1. Set room `max_participants = 2`
2. Already have 2 participants joined

**Request:** `POST {{BASE_URL}}/rooms/{{ROOM_ID}}/participants/{{PARTICIPANT_ID}}/admit`

**Headers:**
```json
Authorization: Bearer {{HOST_TOKEN}}
```

**Expected Response:**
```json
{
  "error": "room_full",
  "message": "room is full"
}
```

**Expected Status:** `409 Conflict`

---

### **Test 5: Invalid Participant ID**

**Request:** `POST {{BASE_URL}}/rooms/{{ROOM_ID}}/participants/invalid-uuid/admit`

**Headers:**
```json
Authorization: Bearer {{HOST_TOKEN}}
```

**Expected Response:**
```json
{
  "error": "invalid_participant_id",
  "message": "participant ID must be a valid UUID"
}
```

**Expected Status:** `400 Bad Request`

---

## Postman Collection Structure

Tạo collection với cấu trúc sau:

```
📁 Meeting Assistant - Waiting Room
  📁 1. Authentication
    ├─ POST Login as Host
    └─ POST Login as Participant
  📁 2. Room Setup
    └─ POST Create Room (Host)
  📁 3. Waiting Room Flow
    ├─ POST Join Room (Participant) → Enter Waiting
    ├─ GET View Waiting List (Host)
    ├─ POST Admit Participant (Host)
    └─ POST Deny Participant (Host)
  📁 4. Verification
    └─ GET List All Participants
  📁 5. Error Cases
    ├─ GET Waiting List (Non-Host) → 403
    ├─ POST Admit (Non-Host) → 403
    ├─ POST Admit Already Joined → 400
    ├─ POST Admit to Full Room → 409
    └─ POST Admit Invalid UUID → 400
```

---

## Automated Tests với Postman Scripts

### Pre-request Script (Collection Level)

```javascript
// Set timestamp
pm.environment.set("timestamp", new Date().getTime());
```

### Test Script cho Login Requests

```javascript
// Save token to environment
if (pm.response.code === 200) {
    const response = pm.response.json();
    
    // Determine if this is host or participant
    const email = JSON.parse(pm.request.body.raw).email;
    
    if (email.includes('host')) {
        pm.environment.set("HOST_TOKEN", response.access_token);
        console.log("✅ Host token saved");
    } else {
        pm.environment.set("PARTICIPANT_TOKEN", response.access_token);
        console.log("✅ Participant token saved");
    }
}
```

### Test Script cho Create Room

```javascript
if (pm.response.code === 200) {
    const response = pm.response.json();
    pm.environment.set("ROOM_ID", response.room.id);
    console.log("✅ Room ID saved:", response.room.id);
}

pm.test("Room created successfully", function () {
    pm.response.to.have.status(200);
    pm.expect(pm.response.json().room).to.have.property('id');
});
```

### Test Script cho Join Room (Waiting)

```javascript
if (pm.response.code === 200) {
    const response = pm.response.json();
    pm.environment.set("PARTICIPANT_ID", response.participant.id);
    console.log("✅ Participant ID saved:", response.participant.id);
}

pm.test("Participant enters waiting room", function () {
    pm.response.to.have.status(200);
    pm.expect(pm.response.json().participant.status).to.eql("waiting");
});
```

### Test Script cho Get Waiting List

```javascript
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

pm.test("Returns waiting participants", function () {
    const response = pm.response.json();
    pm.expect(response.participants).to.be.an('array');
    pm.expect(response.participants.length).to.be.greaterThan(0);
});

pm.test("All participants have waiting status", function () {
    const participants = pm.response.json().participants;
    participants.forEach(p => {
        pm.expect(p.status).to.eql("waiting");
    });
});
```

### Test Script cho Admit Participant

```javascript
pm.test("Participant admitted successfully", function () {
    pm.response.to.have.status(200);
    pm.expect(pm.response.json().message).to.include("admitted");
});
```

---

## Running the Collection

### Manually
1. Import collection vào Postman
2. Select environment
3. Run requests theo thứ tự 1 → 7

### Automatically (Collection Runner)
1. Click **Runner** 
2. Select collection
3. Select environment
4. Set iterations = 1
5. Click **Run**

---

## Expected Full Flow Timeline

```
1. Host Login           → Get HOST_TOKEN
2. Participant Login    → Get PARTICIPANT_TOKEN  
3. Host Creates Room    → Get ROOM_ID
4. Participant Joins    → status='waiting', Get PARTICIPANT_ID
5. Host Views Waiting   → See 1 participant waiting
6. Host Admits         → Participant status='joined' ✅
   OR
   Host Denies         → Participant status='denied' ❌
7. Verify Status       → Check final participant status
```

---

## Troubleshooting

### Issue: "user not authenticated"
- ✅ Check token is valid
- ✅ Token format: `Bearer <token>`
- ✅ Token not expired

### Issue: "room not found"
- ✅ ROOM_ID is set correctly
- ✅ Room exists in database

### Issue: "participant not found"
- ✅ PARTICIPANT_ID is correct
- ✅ Participant record exists

### Issue: Empty waiting list
- ✅ Participant has status='waiting'
- ✅ Check database directly: `SELECT * FROM participants WHERE status='waiting'`

---

## Next Steps

1. **Implement JoinRoom Update**: Modify JoinRoom handler to auto-set status='waiting' for non-host users
2. **Add WebSocket Notifications**: Notify participant when admitted/denied
3. **Add Timeout Logic**: Auto-deny after X minutes in waiting room
4. **Add Bulk Operations**: Admit all, deny all endpoints

---

## Database Queries for Verification

```sql
-- Check participant status
SELECT id, user_id, room_id, status, role, created_at 
FROM participants 
WHERE room_id = 'your-room-uuid';

-- Check waiting participants
SELECT p.id, u.name, u.email, p.status, p.created_at
FROM participants p
JOIN users u ON p.user_id = u.id
WHERE p.room_id = 'your-room-uuid' AND p.status = 'waiting';

-- Check room participant count
SELECT id, name, current_participants, max_participants 
FROM rooms 
WHERE id = 'your-room-uuid';
```

---

## Notes

⚠️ **Important:** Hiện tại `JoinRoom` handler chưa có logic tự động đưa user vào waiting room. Bạn cần update code để:

```go
// In JoinRoom handler
if room.HostID != input.UserID {
    // Non-host users go to waiting room
    participant.Status = entities.ParticipantStatusWaiting
} else {
    // Host joins directly
    participant.Status = entities.ParticipantStatusJoined
}
```

Sau khi update, test flow sẽ hoạt động hoàn chỉnh.
