from fastapi import FastAPI
import uvicorn
import os
import pymysql
from datetime import datetime
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="StreamFlow Billing Service", version="1.0.0")

DB_CONFIG = {
    "host": os.getenv("DB_HOST", "localhost"),
    "port": int(os.getenv("DB_PORT", "3306")),
    "database": os.getenv("DB_NAME", "billing_db"),
    "user": os.getenv("DB_USER", "root"),
    "password": os.getenv("DB_PASSWORD", "password"),
}

def get_db_connection():
    try:
        conn = pymysql.connect(**DB_CONFIG)
        return conn
    except pymysql.Error as e:
        logger.error(f"Error conectando a la base de datos: {e}")
        raise Exception("Error de conexión a la base de datos")

def init_db():
    conn = get_db_connection()
    cursor = conn.cursor()
    
    create_table_query = """
    CREATE TABLE IF NOT EXISTS invoices (
        id INT AUTO_INCREMENT PRIMARY KEY,
        user_id INT NOT NULL,
        amount DECIMAL(10, 2) NOT NULL,
        issue_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        payment_date TIMESTAMP NULL,
        status ENUM('Pendiente', 'Pagado', 'Vencido') NOT NULL DEFAULT 'Pendiente',
        deleted_at TIMESTAMP NULL,
        INDEX idx_user_id (user_id),
        INDEX idx_status (status),
        INDEX idx_deleted (deleted_at)
    )
    """
    
    cursor.execute(create_table_query)
    conn.commit()
    cursor.close()
    conn.close()
    
    logger.info("Billing database initialized")

@app.post("/invoices")
async def create_invoice(invoice_data: dict):
    """Crear una nueva factura"""
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        
        query = """
        INSERT INTO invoices (user_id, amount, status)
        VALUES (%s, %s, %s)
        """
        
        cursor.execute(query, (
            invoice_data.get("user_id"),
            invoice_data.get("amount"),
            invoice_data.get("status", "Pendiente")
        ))
        
        invoice_id = cursor.lastrowid
        conn.commit()
        
        # Obtener la factura creada
        cursor.execute("SELECT * FROM invoices WHERE id = %s", (invoice_id,))
        invoice = cursor.fetchone()
        
        cursor.close()
        conn.close()
        
        return {
            "id": invoice[0],
            "user_id": invoice[1],
            "amount": float(invoice[2]),
            "issue_date": invoice[3].isoformat(),
            "payment_date": invoice[4].isoformat() if invoice[4] else None,
            "status": invoice[5]
        }
        
    except Exception as e:
        logger.error(f"Error creando factura: {e}")
        return {"error": str(e)}

@app.get("/invoices/{invoice_id}")
async def get_invoice(invoice_id: int):
    """Obtener una factura por ID"""
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        
        cursor.execute(
            "SELECT * FROM invoices WHERE id = %s AND deleted_at IS NULL",
            (invoice_id,)
        )
        invoice = cursor.fetchone()
        
        cursor.close()
        conn.close()
        
        if not invoice:
            return {"error": "Factura no encontrada"}
        
        return {
            "id": invoice[0],
            "user_id": invoice[1],
            "amount": float(invoice[2]),
            "issue_date": invoice[3].isoformat(),
            "payment_date": invoice[4].isoformat() if invoice[4] else None,
            "status": invoice[5]
        }
        
    except Exception as e:
        logger.error(f"Error obteniendo factura: {e}")
        return {"error": str(e)}

@app.patch("/invoices/{invoice_id}")
async def update_invoice_status(invoice_id: int, update_data: dict):
    """Actualizar estado de una factura"""
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        
        new_status = update_data.get("status")
        payment_date = None
        
        if new_status == "Pagado":
            payment_date = datetime.now()
        
        if payment_date:
            cursor.execute(
                "UPDATE invoices SET status = %s, payment_date = %s WHERE id = %s AND deleted_at IS NULL",
                (new_status, payment_date, invoice_id)
            )
        else:
            cursor.execute(
                "UPDATE invoices SET status = %s WHERE id = %s AND deleted_at IS NULL",
                (new_status, invoice_id)
            )
        
        conn.commit()
        
        # Obtener factura actualizada
        cursor.execute("SELECT * FROM invoices WHERE id = %s", (invoice_id,))
        invoice = cursor.fetchone()
        
        cursor.close()
        conn.close()
        
        if not invoice:
            return {"error": "Factura no encontrada"}
        
        return {
            "id": invoice[0],
            "user_id": invoice[1],
            "amount": float(invoice[2]),
            "issue_date": invoice[3].isoformat(),
            "payment_date": invoice[4].isoformat() if invoice[4] else None,
            "status": invoice[5]
        }
        
    except Exception as e:
        logger.error(f"Error actualizando factura: {e}")
        return {"error": str(e)}

@app.delete("/invoices/{invoice_id}")
async def delete_invoice(invoice_id: int):
    """Eliminar una factura (soft delete)"""
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        
        # Verificar si la factura está pagada
        cursor.execute(
            "SELECT status FROM invoices WHERE id = %s AND deleted_at IS NULL",
            (invoice_id,)
        )
        result = cursor.fetchone()
        
        if not result:
            cursor.close()
            conn.close()
            return {"error": "Factura no encontrada"}
        
        if result[0] == "Pagado":
            cursor.close()
            conn.close()
            return {"error": "No se puede eliminar una factura pagada"}
        
        # Soft delete
        cursor.execute(
            "UPDATE invoices SET deleted_at = NOW() WHERE id = %s",
            (invoice_id,)
        )
        
        conn.commit()
        cursor.close()
        conn.close()
        
        return {"message": "Factura eliminada exitosamente"}
        
    except Exception as e:
        logger.error(f"Error eliminando factura: {e}")
        return {"error": str(e)}

@app.get("/invoices")
async def list_invoices(user_id: int = None, status: str = None):
    """Listar facturas con filtros opcionales"""
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        
        query = "SELECT * FROM invoices WHERE deleted_at IS NULL"
        params = []
        
        if user_id:
            query += " AND user_id = %s"
            params.append(user_id)
        
        if status:
            query += " AND status = %s"
            params.append(status)
        
        query += " ORDER BY issue_date DESC"
        
        cursor.execute(query, params)
        invoices = cursor.fetchall()
        
        cursor.close()
        conn.close()
        
        result = []
        for invoice in invoices:
            result.append({
                "id": invoice[0],
                "user_id": invoice[1],
                "amount": float(invoice[2]),
                "issue_date": invoice[3].isoformat(),
                "payment_date": invoice[4].isoformat() if invoice[4] else None,
                "status": invoice[5]
            })
        
        return {"invoices": result}
        
    except Exception as e:
        logger.error(f"Error listando facturas: {e}")
        return {"error": str(e)}

@app.get("/health")
async def health_check():
    return {"status": "healthy", "service": "billing"}

@app.on_event("startup")
async def startup_event():
    init_db()
    logger.info("Servicio de facturación iniciado")

if __name__ == "__main__":
    port = int(os.getenv("PORT", 50052))
    uvicorn.run(app, host="0.0.0.0", port=port)
