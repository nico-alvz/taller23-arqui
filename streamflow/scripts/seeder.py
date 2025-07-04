#!/usr/bin/env python3
"""
Seeder para poblar las bases de datos de StreamFlow con datos de prueba
"""

import requests
import random
import time
import json
from datetime import datetime, timedelta

# Configuración
API_BASE_URL = "https://localhost"  # Usar HTTPS a través de Nginx
AUTH_API_URL = "http://localhost:8001"  # Auth service directo para bootstrap

# Datos de prueba
FIRST_NAMES = [
    "Carlos", "María", "José", "Ana", "Luis", "Elena", "Pedro", "Sofia", "Miguel", "Carmen",
    "Antonio", "Isabel", "Francisco", "Pilar", "Manuel", "Teresa", "David", "Lucia", "Javier", "Rosa",
    "Rafael", "Cristina", "Daniel", "Patricia", "Jorge", "Marta", "Alejandro", "Beatriz", "Fernando", "Alicia"
]

LAST_NAMES = [
    "García", "Rodríguez", "González", "Fernández", "López", "Martínez", "Sánchez", "Pérez", "Gómez", "Martín",
    "Jiménez", "Ruiz", "Hernández", "Díaz", "Moreno", "Muñoz", "Álvarez", "Romero", "Alonso", "Gutiérrez",
    "Navarro", "Torres", "Domínguez", "Vázquez", "Ramos", "Gil", "Ramírez", "Serrano", "Blanco", "Suárez"
]

VIDEO_TITLES = [
    "Aventuras en el Espacio", "El Misterio de la Casa Vieja", "Comedia en la Oficina", "Documentos Perdidos",
    "La Última Batalla", "Romance en París", "Thriller Nocturno", "Ciencia Ficción 2024", "Drama Familiar",
    "Acción Extrema", "Horror en el Bosque", "Comedia Romántica", "Historia de Guerra", "Fantasía Épica",
    "Documental Naturaleza", "Suspense Psicológico", "Musical Broadway", "Western Clásico", "Animación 3D",
    "Biografía Inspiradora", "Serie de Detectives", "Aventura Submarina", "Comedia Stand-up", "Drama Legal",
    "Ciencia y Tecnología", "Viajes por el Mundo", "Cocina Internacional", "Deportes Extremos", "Arte Contemporáneo",
    "Historia Antigua", "Música Clásica", "Baile Moderno", "Teatro Experimental", "Cine Independiente"
]

VIDEO_DESCRIPTIONS = [
    "Una emocionante aventura que te llevará a los confines del universo",
    "Misterio y suspense en una historia que te mantendrá en vilo",
    "Risas garantizadas con esta divertida comedia",
    "Un thriller que te hará reflexionar sobre la naturaleza humana",
    "Acción y aventura en estado puro",
    "Una historia de amor que trasciende el tiempo",
    "Terror psicológico que te pondrá los pelos de punta",
    "Ciencia ficción de última generación con efectos espectaculares",
    "Un drama conmovedor sobre las relaciones familiares",
    "Documental educativo y entretenido"
]

VIDEO_GENRES = [
    "Acción", "Comedia", "Drama", "Terror", "Ciencia Ficción", "Romance", "Thriller", "Documental",
    "Aventura", "Fantasía", "Musical", "Western", "Animación", "Biografía", "Historia", "Deportes"
]

COMMENTS = [
    "¡Excelente video! Me encantó",
    "Muy buena calidad, recomendado",
    "Interesante contenido, gracias por compartir",
    "¡Increíble! Lo volveré a ver",
    "Buen trabajo, sigue así",
    "Me gustó mucho, muy entretenido",
    "Excelente producción y edición",
    "Contenido de alta calidad",
    "¡Fantástico! Esperando más videos así",
    "Muy bien explicado y presentado"
]

class StreamFlowSeeder:
    def __init__(self):
        self.admin_token = None
        self.users = []
        self.videos = []
        self.invoices = []
    
    def login_admin(self):
        """Iniciar sesión como administrador"""
        print("🔐 Iniciando sesión como administrador...")
        
        login_data = {
            "email": "admin@streamflow.com",
            "password": "admin123"
        }
        
        try:
            response = requests.post(f"{AUTH_API_URL}/auth/login", json=login_data, verify=False)
            if response.status_code == 200:
                data = response.json()
                self.admin_token = data["access_token"]
                print("✅ Sesión de administrador iniciada")
                return True
            else:
                print(f"❌ Error iniciando sesión: {response.text}")
                return False
        except Exception as e:
            print(f"❌ Error conectando al servicio de auth: {e}")
            return False
    
    def create_users(self, count=150):
        """Crear usuarios de prueba"""
        print(f"👥 Creando {count} usuarios...")
        
        headers = {"Authorization": f"Bearer {self.admin_token}"} if self.admin_token else {}
        
        for i in range(count):
            first_name = random.choice(FIRST_NAMES)
            last_name = random.choice(LAST_NAMES)
            email = f"{first_name.lower()}.{last_name.lower()}.{i}@streamflow.com"
            role = "Cliente" if i < count - 10 else "Administrador"  # Últimos 10 como admin
            
            user_data = {
                "first_name": first_name,
                "last_name": last_name,
                "email": email,
                "password": "password123",
                "confirm_password": "password123",
                "role": role
            }
            
            try:
                response = requests.post(f"{API_BASE_URL}/usuarios", json=user_data, headers=headers, verify=False)
                if response.status_code in [200, 201]:
                    user = response.json()
                    self.users.append(user)
                    print(f"  ✅ Usuario creado: {email} ({role})")
                else:
                    print(f"  ❌ Error creando usuario {email}: {response.text}")
            except Exception as e:
                print(f"  ❌ Error creando usuario {email}: {e}")
            
            time.sleep(0.1)  # Para no sobrecargar
        
        print(f"✅ {len(self.users)} usuarios creados")
    
    def create_videos(self, count=500):
        """Crear videos de prueba"""
        print(f"🎥 Creando {count} videos...")
        
        headers = {"Authorization": f"Bearer {self.admin_token}"}
        
        for i in range(count):
            title = f"{random.choice(VIDEO_TITLES)} #{i+1}"
            description = random.choice(VIDEO_DESCRIPTIONS)
            genre = random.choice(VIDEO_GENRES)
            
            video_data = {
                "title": title,
                "description": description,
                "genre": genre
            }
            
            try:
                response = requests.post(f"{API_BASE_URL}/videos", json=video_data, headers=headers, verify=False)
                if response.status_code in [200, 201]:
                    video = response.json()
                    self.videos.append(video)
                    print(f"  ✅ Video creado: {title}")
                else:
                    print(f"  ❌ Error creando video {title}: {response.text}")
            except Exception as e:
                print(f"  ❌ Error creando video {title}: {e}")
            
            time.sleep(0.1)
        
        print(f"✅ {len(self.videos)} videos creados")
    
    def create_invoices(self, count=350):
        """Crear facturas de prueba"""
        print(f"🧾 Creando {count} facturas...")
        
        headers = {"Authorization": f"Bearer {self.admin_token}"}
        statuses = ["Pendiente", "Pagado", "Vencido"]
        
        for i in range(count):
            if not self.users:
                print("❌ No hay usuarios para crear facturas")
                break
            
            user = random.choice(self.users)
            amount = round(random.uniform(9.99, 99.99), 2)
            status = random.choice(statuses)
            
            invoice_data = {
                "user_id": user.get("id", 1),
                "amount": amount,
                "status": status
            }
            
            try:
                response = requests.post(f"{API_BASE_URL}/facturas", json=invoice_data, headers=headers, verify=False)
                if response.status_code in [200, 201]:
                    invoice = response.json()
                    self.invoices.append(invoice)
                    print(f"  ✅ Factura creada: ${amount} para usuario {user.get('email', 'N/A')}")
                else:
                    print(f"  ❌ Error creando factura: {response.text}")
            except Exception as e:
                print(f"  ❌ Error creando factura: {e}")
            
            time.sleep(0.1)
        
        print(f"✅ {len(self.invoices)} facturas creadas")
    
    def create_likes(self, count=75):
        """Crear likes de prueba"""
        print(f"👍 Creando {count} likes...")
        
        # Crear likes con diferentes usuarios
        for i in range(count):
            if not self.users or not self.videos:
                print("❌ No hay suficientes usuarios o videos para crear likes")
                break
            
            user = random.choice(self.users)
            video = random.choice(self.videos)
            
            # Simular login del usuario
            login_data = {
                "email": user.get("email"),
                "password": "password123"
            }
            
            try:
                # Login
                auth_response = requests.post(f"{AUTH_API_URL}/auth/login", json=login_data, verify=False)
                if auth_response.status_code != 200:
                    continue
                
                user_token = auth_response.json()["access_token"]
                headers = {"Authorization": f"Bearer {user_token}"}
                
                # Dar like
                video_id = video.get("id", 1)
                response = requests.post(f"{API_BASE_URL}/interacciones/{video_id}/likes", headers=headers, verify=False)
                if response.status_code in [200, 201]:
                    print(f"  ✅ Like creado para video '{video.get('title', 'N/A')}'")
                else:
                    print(f"  ❌ Error creando like: {response.text}")
            except Exception as e:
                print(f"  ❌ Error creando like: {e}")
            
            time.sleep(0.1)
        
        print(f"✅ Likes creados")
    
    def create_comments(self, count=35):
        """Crear comentarios de prueba"""
        print(f"💬 Creando {count} comentarios...")
        
        for i in range(count):
            if not self.users or not self.videos:
                print("❌ No hay suficientes usuarios o videos para crear comentarios")
                break
            
            user = random.choice(self.users)
            video = random.choice(self.videos)
            comment_text = random.choice(COMMENTS)
            
            # Simular login del usuario
            login_data = {
                "email": user.get("email"),
                "password": "password123"
            }
            
            try:
                # Login
                auth_response = requests.post(f"{AUTH_API_URL}/auth/login", json=login_data, verify=False)
                if auth_response.status_code != 200:
                    continue
                
                user_token = auth_response.json()["access_token"]
                headers = {"Authorization": f"Bearer {user_token}"}
                
                # Crear comentario
                video_id = video.get("id", 1)
                comment_data = {"comment": comment_text}
                response = requests.post(f"{API_BASE_URL}/interacciones/{video_id}/comentarios", json=comment_data, headers=headers, verify=False)
                if response.status_code in [200, 201]:
                    print(f"  ✅ Comentario creado para video '{video.get('title', 'N/A')}'")
                else:
                    print(f"  ❌ Error creando comentario: {response.text}")
            except Exception as e:
                print(f"  ❌ Error creando comentario: {e}")
            
            time.sleep(0.1)
        
        print(f"✅ Comentarios creados")
    
    def run(self):
        """Ejecutar el seeder completo"""
        print("🌱 Iniciando seeder de StreamFlow...")
        print("Esperando que los servicios estén listos...")
        time.sleep(10)  # Esperar a que los servicios estén listos
        
        if not self.login_admin():
            print("❌ No se pudo iniciar sesión como administrador")
            return
        
        # Crear datos en orden
        self.create_users(150)
        self.create_videos(500)
        self.create_invoices(350)
        self.create_likes(75)
        self.create_comments(35)
        
        print("🎉 Seeder completado exitosamente!")
        print(f"📊 Resumen:")
        print(f"  - {len(self.users)} usuarios")
        print(f"  - {len(self.videos)} videos")
        print(f"  - {len(self.invoices)} facturas")

if __name__ == "__main__":
    import urllib3
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    
    seeder = StreamFlowSeeder()
    seeder.run()
