from fastapi import APIRouter, FastAPI, HTTPException, Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from pydantic import BaseModel, EmailStr, Extra
from typing import Optional
import psycopg2
import psycopg2.extras
import jwt
import bcrypt
import os
from datetime import datetime, timedelta
import logging
import pika
import json

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="StreamFlow Auth Service", version="1.0.0")
security = HTTPBearer()

# Configuraciones
SECRET_KEY = os.getenv("JWT_SECRET_KEY", "streamflow_secret_key_2024")
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 1440

DB_CONFIG = {
    "host": os.getenv("DB_HOST", "localhost"),
    "port": os.getenv("DB_PORT", "5432"),
    "database": os.getenv("DB_NAME", "auth_db"),
    "user": os.getenv("DB_USER", "postgres"),
    "password": os.getenv("DB_PASSWORD", "password"),
}

RABBITMQ_HOST = os.getenv("RABBITMQ_HOST", "rabbitmq")
RABBITMQ_USER = os.getenv("RABBITMQ_USER", "admin")
RABBITMQ_PASS = os.getenv("RABBITMQ_PASS", "password")
RABBITMQ_QUEUE = os.getenv("RABBITMQ_QUEUE", "monitoring")

# Modelos
class LoginRequest(BaseModel):
    email: EmailStr
    password: str

class ChangePasswordRequest(BaseModel):
    current_password: str
    new_password: str
    confirm_new_password: str

class UserResponse(BaseModel):
    id: int
    first_name: str
    last_name: str
    email: str
    role: str
    created_at: str

class LoginResponse(BaseModel):
    user: UserResponse
    access_token: str
    token_type: str = "bearer"

# RabbitMQ Publisher
def publish_event(event_type: str, payload: dict):
    credentials = pika.PlainCredentials(RABBITMQ_USER, RABBITMQ_PASS)
    parameters = pika.ConnectionParameters(host=RABBITMQ_HOST, credentials=credentials)
    try:
        connection = pika.BlockingConnection(parameters)
        channel = connection.channel()
        channel.queue_declare(queue=RABBITMQ_QUEUE, durable=True)
        message = {
            "type": event_type,
            "timestamp": datetime.utcnow().isoformat(),
            "data": payload
        }
        channel.basic_publish(
            exchange='',
            routing_key=RABBITMQ_QUEUE,
            body=json.dumps(message),
            properties=pika.BasicProperties(delivery_mode=2)
        )
        connection.close()
        logger.info(f"📤 Evento publicado a RabbitMQ: {event_type}")
    except Exception as e:
        logger.error(f"❌ Error al publicar mensaje en RabbitMQ: {e}")

# Base de datos
def get_db_connection():
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        conn.autocommit = True
        return conn
    except psycopg2.Error as e:
        logger.error(f"Error conectando a la base de datos: {e}")
        raise HTTPException(status_code=500, detail="Error de conexión a la base de datos")

def init_db():
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS users (
                id SERIAL PRIMARY KEY,
                first_name VARCHAR(50) NOT NULL,
                last_name VARCHAR(50) NOT NULL,
                email VARCHAR(100) NOT NULL UNIQUE,
                password VARCHAR(255) NOT NULL,
                role VARCHAR(20) NOT NULL CHECK (role IN ('Administrador', 'Cliente')),
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                deleted_at TIMESTAMP NULL
            )
        """)
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS token_blacklist (
                id SERIAL PRIMARY KEY,
                jti VARCHAR(255) NOT NULL UNIQUE,
                user_id INTEGER NOT NULL,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)
        admin_password = bcrypt.hashpw("admin123".encode("utf-8"), bcrypt.gensalt()).decode("utf-8")
        cursor.execute("""
            INSERT INTO users (first_name, last_name, email, password, role)
            VALUES (%s, %s, %s, %s, %s)
            ON CONFLICT (email) DO NOTHING
        """, ("Admin", "StreamFlow", "admin@streamflow.com", admin_password, "Administrador"))
        logger.info("✅ Base de datos inicializada correctamente.")
    except Exception as e:
        logger.error(f"❌ Error al inicializar la base de datos: {e}")
        raise
    finally:
        cursor.close()
        conn.close()

# JWT
def create_access_token(user_id: int, email: str, role: str):
    expire = datetime.utcnow() + timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES)
    jti = f"{user_id}_{datetime.utcnow().timestamp()}"
    payload = {"sub": str(user_id), "email": email, "role": role, "exp": expire, "jti": jti}
    return jwt.encode(payload, SECRET_KEY, algorithm=ALGORITHM), jti

def verify_token(token: str):
    try:
        return jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
    except jwt.ExpiredSignatureError:
        raise HTTPException(status_code=401, detail="Token expirado")
    except jwt.InvalidTokenError:
        raise HTTPException(status_code=401, detail="Token inválido")

def is_token_blacklisted(jti: str):
    conn = get_db_connection()
    cursor = conn.cursor()
    cursor.execute("SELECT id FROM token_blacklist WHERE jti = %s", (jti,))
    result = cursor.fetchone()
    cursor.close()
    conn.close()
    return result is not None

def get_current_user(credentials: HTTPAuthorizationCredentials = Depends(security)):
    token = credentials.credentials
    payload = verify_token(token)
    jti = payload.get("jti")
    if is_token_blacklisted(jti):
        raise HTTPException(status_code=401, detail="Token invalidado")
    return {
        "id": int(payload.get("sub")),
        "email": payload.get("email"),
        "role": payload.get("role"),
        "jti": jti
    }

# Rutas
@app.post("/auth/login", response_model=LoginResponse)
async def login(login_data: LoginRequest):
    conn = get_db_connection()
    cursor = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
    try:
        cursor.execute("SELECT * FROM users WHERE email = %s AND deleted_at IS NULL", (login_data.email,))
        user = cursor.fetchone()
        if not user or not bcrypt.checkpw(login_data.password.encode(), user["password"].encode()):
            raise HTTPException(status_code=401, detail="Credenciales inválidas")

        token, _ = create_access_token(user["id"], user["email"], user["role"])
        user_data = UserResponse(
            id=user["id"],
            first_name=user["first_name"],
            last_name=user["last_name"],
            email=user["email"],
            role=user["role"],
            created_at=user["created_at"].isoformat()
        )

        publish_event("USER_LOGIN", {
            "user_id": user["id"],
            "email": user["email"],
            "role": user["role"]
        })

        return LoginResponse(user=user_data, access_token=token)
    finally:
        cursor.close()
        conn.close()

@app.patch("/auth/users/{user_id}")
async def change_password(user_id: int, password_data: ChangePasswordRequest, current_user: dict = Depends(get_current_user)):
    if current_user["role"] != "Administrador" and current_user["id"] != user_id:
        raise HTTPException(status_code=403, detail="No tiene permisos para esta acción")
    if password_data.new_password != password_data.confirm_new_password:
        raise HTTPException(status_code=400, detail="Las contraseñas no coinciden")

    conn = get_db_connection()
    cursor = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
    try:
        cursor.execute("SELECT * FROM users WHERE id = %s AND deleted_at IS NULL", (user_id,))
        user = cursor.fetchone()
        if not user:
            raise HTTPException(status_code=404, detail="Usuario no encontrado")

        if current_user["id"] == user_id:
            if not bcrypt.checkpw(password_data.current_password.encode(), user["password"].encode()):
                raise HTTPException(status_code=400, detail="Contraseña actual incorrecta")

        new_hash = bcrypt.hashpw(password_data.new_password.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")
        cursor.execute("UPDATE users SET password = %s WHERE id = %s", (new_hash, user_id))
        conn.commit()

        publish_event("USER_PWD_CHANGED", {
            "user_id": current_user["id"],
            "email": current_user["email"],
            "role": current_user["role"]
        })

        return {"message": "Contraseña actualizada exitosamente"}
    finally:
        cursor.close()
        conn.close()

@app.post("/auth/logout")
async def logout(current_user: dict = Depends(get_current_user)):
    conn = get_db_connection()
    cursor = conn.cursor()
    try:
        cursor.execute("INSERT INTO token_blacklist (jti, user_id) VALUES (%s, %s)", (current_user["jti"], current_user["id"]))
        conn.commit()

        publish_event("USER_LOGOUT", {
            "user_id": current_user["id"],
            "email": current_user["email"],
            "role": current_user["role"]
        })

        return {"message": "Sesión cerrada exitosamente"}
    finally:
        cursor.close()
        conn.close()

@app.get("/health")
async def health_check():
    return {"status": "healthy", "service": "auth"}

@app.on_event("startup")
async def startup_event():
    init_db()
    logger.info("✅ Servicio de autenticación iniciado")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("main:app", host="0.0.0.0", port=8001)

