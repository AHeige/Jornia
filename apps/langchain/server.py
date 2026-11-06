"""A bounded LangChain model step; no agent loop, memory, filesystem tools, or tracing."""
import asyncio
import logging
import os
import re
from typing import Literal

os.environ['LANGSMITH_TRACING'] = 'false'
os.environ['LANGCHAIN_TRACING_V2'] = 'false'
from fastapi import FastAPI, HTTPException
from langchain_core.messages import HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI
from pydantic import BaseModel, ConfigDict, Field

app = FastAPI(docs_url=None, redoc_url=None)
slots = asyncio.Semaphore(2)
MODEL = os.getenv('BERGET_MODEL', 'mistralai/Mistral-Small-3.2-24B-Instruct-2506')

class Message(BaseModel):
    model_config = ConfigDict(extra='forbid')
    role: Literal['system', 'user']
    content: str = Field(max_length=50000)

class Request(BaseModel):
    model_config = ConfigDict(extra='forbid')
    model: str
    messages: list[Message] = Field(min_length=1, max_length=3)
    max_tokens: int = Field(default=512, ge=1, le=512)
    temperature: float = Field(default=0, ge=0, le=1)
    response_format: dict[str, str]

@app.get('/healthz')
def health():
    return {'status': 'ok', 'inference_configured': bool(os.getenv('BERGET_API_KEY'))}

@app.post('/v1/chat/completions')
async def completion(request: Request):
    if request.model != MODEL or request.response_format != {'type': 'json_object'}:
        raise HTTPException(400, 'Unsupported model or format')
    if not os.getenv('BERGET_API_KEY'):
        raise HTTPException(503, 'Inference is not configured')
    try:
        model = ChatOpenAI(
            model=MODEL,
            api_key=os.environ['BERGET_API_KEY'],
            base_url=os.getenv('BERGET_BASE_URL', 'https://api.berget.ai/v1'),
            temperature=request.temperature, max_tokens=request.max_tokens,
            timeout=30, max_retries=0, stream_usage=False,
            model_kwargs={'response_format': {'type': 'json_object'}},
        )
        messages = [SystemMessage(m.content) if m.role == 'system' else HumanMessage(m.content) for m in request.messages]
        question = request.messages[-1].content.rsplit('\nQuestion:\n', 1)[-1].strip()
        if re.match(r'(?i)^(var|vem|vad|vilka|hur|berätta|kan du|hjälp)\b', question):
            # Explicit language control without another paid classification/translation call.
            messages.insert(0, SystemMessage('Svara på svenska. Skriv hela fältet text på svenska, men behåll spelets egennamn och evidence_ids oförändrade.'))
        async with slots:
            result = await model.ainvoke(messages, config={'callbacks': []})
        if not isinstance(result.content, str):
            raise ValueError('Expected text')
        metadata = result.response_metadata
        response = {'model': metadata.get('model_name') or MODEL,
                    'choices': [{'finish_reason': metadata.get('finish_reason'), 'message': {'content': result.content}}]}
        usage = result.usage_metadata
        if usage and 'input_tokens' in usage and 'output_tokens' in usage:
            response['usage'] = {'prompt_tokens': usage['input_tokens'], 'completion_tokens': usage['output_tokens']}
        return response
    except Exception as error:
        # Operational metadata only. Never log exception strings, prompts,
        # provider bodies or credentials (SDK errors can embed request data).
        logging.warning('model_step_failed kind=%s upstream_status=%s',
                        type(error).__name__, getattr(error, 'status_code', None))
        # No request content, responses, credentials or upstream diagnostics in logs/errors.
        raise HTTPException(502, 'Model step failed') from None
