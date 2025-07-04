import grpc
from concurrent import futures
import time
import logging

import email_pb2
import email_pb2_grpc
from main import send_email  # Reutilizamos la lógica ya escrita

class EmailServiceServicer(email_pb2_grpc.EmailServiceServicer):

    def SendWelcomeEmail(self, request, context):
        subject = "¡Bienvenido a StreamFlow!"
        body = f"""
        <html><body>
            <h2>¡Hola {request.name}!</h2>
            <p>Gracias por unirte a StreamFlow.</p>
        </body></html>
        """
        success = send_email(request.email, subject, body)
        return email_pb2.EmailResponse(success=success, message="Correo de bienvenida enviado")

    def SendInvoiceUpdateEmail(self, request, context):
        subject = f"Actualización de Factura #{request.invoice_id}"
        body = f"""
        <html><body>
            <p>Factura: {request.invoice_id}</p>
            <p>Monto: ${request.amount}</p>
            <p>Estado: {request.status}</p>
        </body></html>
        """
        success = send_email(request.user_email, subject, body)
        return email_pb2.EmailResponse(success=success, message="Correo de factura enviado")

    def SendPasswordUpdatedEmail(self, request, context):
        subject = "Tu contraseña ha sido actualizada"
        body = f"""
        <html><body>
            <p>Hola {request.user_name},</p>
            <p>Si no realizaste este cambio, contacta soporte.</p>
        </body></html>
        """
        success = send_email(request.user_email, subject, body)
        return email_pb2.EmailResponse(success=success, message="Correo de contraseña enviado")

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    email_pb2_grpc.add_EmailServiceServicer_to_server(EmailServiceServicer(), server)
    server.add_insecure_port('[::]:50051')
    logging.info("gRPC server iniciado en puerto 50051")
    server.start()
    server.wait_for_termination()
