import io
import qrcode
from fastapi import FastAPI, Header, HTTPException
from fastapi.responses import StreamingResponse
from fastapi.middleware.cors import CORSMiddleware

app = FastAPI(title="QR Generation Service")


app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/health")
def health():
    return {"status": "ok"}

@app.get("/api/qr/generate")
def generate_qr(data: str):
    """
    Generates a generic QR code image buffer dynamically based on provided text data.
    """
    if not data:
        raise HTTPException(status_code=400, detail="Data query parameter is missing")
    
    
    qr = qrcode.QRCode(
        version=1,
        error_correction=qrcode.constants.ERROR_CORRECT_L,
        box_size=10,
        border=4,
    )
    qr.add_data(data)
    qr.make(fit=True)

    img = qr.make_image(fill_color="black", back_color="white")
    
    
    buf = io.BytesIO()
    img.save(buf, format="PNG")
    buf.seek(0)

    
    return StreamingResponse(buf, media_type="image/png")
