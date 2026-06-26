import os
import json
import re
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from dotenv import load_dotenv
import google.generativeai as genai

load_dotenv()

# Configure Gemini
genai.configure(api_key=os.getenv("GEMINI_API_KEY"))
model = genai.GenerativeModel("gemini-2.0-flash")

app = FastAPI(title="Pikyon AI Service", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

# ============================================================
# MODELS
# ============================================================

class AnalyzeRequest(BaseModel):
    title: str
    story: str
    lang: str = "en"

class AIAnalysisResult(BaseModel):
    vibe: str
    vibe_reason: str
    tone: str
    captions: dict
    suggested_tags: list[str]

class SongRecommendation(BaseModel):
    title: str
    artist: str
    reason: str
    mood: str
    spotify_search: str

# ============================================================
# HELPERS
# ============================================================

LANGUAGE_MAP = {
    "en": "English",
    "sw": "Swahili",
    "fr": "French",
    "es": "Spanish",
    "ar": "Arabic",
}

def clean_json(text: str) -> str:
    """Remove markdown code blocks from Gemini response"""
    text = re.sub(r"```json\s*", "", text)
    text = re.sub(r"```\s*", "", text)
    return text.strip()

# ============================================================
# ROUTES
# ============================================================

@app.get("/health")
def health():
    return {"status": "ok", "service": "Pikyon AI", "model": "gemini-2.0-flash"}

@app.post("/analyze", response_model=AIAnalysisResult)
async def analyze_memory(req: AnalyzeRequest):
    """
    Analyzes a memory and returns:
    - A song vibe that matches the emotional tone
    - Social media captions
    - Suggested tags
    - Emotional tone
    """
    lang_name = LANGUAGE_MAP.get(req.lang, "English")

    prompt = f"""You are an empathetic AI assistant for Pikyon, a personal digital memoir app.

Analyze this personal memory and return a JSON object ONLY.
No markdown, no explanation, just pure JSON.

Memory Title: "{req.title}"
Memory Story: "{req.story}"
Response Language: {lang_name}

Return EXACTLY this JSON structure:
{{
  "vibe": "<Song Title - Artist Name that perfectly matches this memory's emotional tone>",
  "vibe_reason": "<One warm sentence in {lang_name} explaining why this song fits this memory>",
  "tone": "<exactly one word from: nostalgic, joyful, bittersweet, peaceful, melancholic, triumphant, tender, hopeful, reflective>",
  "captions": {{
    "twitter": "<Engaging tweet in {lang_name} under 280 chars with 2-3 relevant hashtags>",
    "instagram": "<Beautiful Instagram caption in {lang_name} 100-150 chars with 5 hashtags>",
    "linkedin": "<Professional thoughtful reflection in {lang_name} 200-250 chars>"
  }},
  "suggested_tags": ["<3 to 5 relevant single-word tags in English>"]
}}"""

    try:
        response = model.generate_content(prompt)
        cleaned = clean_json(response.text)
        result = json.loads(cleaned)
        return AIAnalysisResult(**result)
    except json.JSONDecodeError as e:
        raise HTTPException(
            status_code=500,
            detail=f"Failed to parse AI response: {str(e)}"
        )
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"AI analysis failed: {str(e)}"
        )

@app.post("/recommend-song", response_model=SongRecommendation)
async def recommend_song(req: AnalyzeRequest):
    """
    Recommends a specific song that matches the memory's mood.
    Returns song details and a Spotify search URL.
    """
    prompt = f"""You are a music expert and therapist for Pikyon memoir app.

Based on this personal memory, recommend ONE perfect song.
Return JSON ONLY, no markdown.

Memory Title: "{req.title}"
Memory Story: "{req.story}"

Return EXACTLY this JSON:
{{
  "title": "<Song title>",
  "artist": "<Artist name>",
  "reason": "<One sentence why this song perfectly captures this memory>",
  "mood": "<one word mood: nostalgic|joyful|bittersweet|peaceful|melancholic|triumphant|tender>",
  "spotify_search": "<spotify:search:Song+Title+Artist+Name>"
}}"""

    try:
        response = model.generate_content(prompt)
        cleaned = clean_json(response.text)
        result = json.loads(cleaned)
        return SongRecommendation(**result)
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Song recommendation failed: {str(e)}"
        )

@app.post("/translate-caption")
async def translate_caption(data: dict):
    """Translates a caption to the target language"""
    caption = data.get("caption", "")
    target_lang = data.get("lang", "en")
    lang_name = LANGUAGE_MAP.get(target_lang, "English")

    prompt = f"""Translate this social media caption to {lang_name}.
Keep hashtags in English. Return only the translated text, nothing else.

Caption: "{caption}"
"""
    try:
        response = model.generate_content(prompt)
        return {"translated": response.text.strip(), "lang": target_lang}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
        