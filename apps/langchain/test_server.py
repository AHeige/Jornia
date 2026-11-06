import asyncio
import json
import os
import unittest
from unittest.mock import patch

import httpx
from langchain_openai import ChatOpenAI as RealChatOpenAI
import server

class ModelStepTest(unittest.TestCase):
    def test_real_langchain_with_fake_provider(self):
        requests=[]
        async def fake(request):
            requests.append(json.loads(request.content))
            self.assertEqual(request.headers['authorization'], 'Bearer unit-test')
            return httpx.Response(200,json={'id':'test-completion','object':'chat.completion','created':1,'model':server.MODEL,
                'choices':[{'index':0,'finish_reason':'stop','message':{'role':'assistant','content':'{"status":"answered","text":"Astranaar is in Ashenvale.","evidence_ids":["entity:astranaar"]}'}}],
                'usage':{'prompt_tokens':100,'completion_tokens':20,'total_tokens':120}})
        async def exercise():
            client=httpx.AsyncClient(transport=httpx.MockTransport(fake))
            def factory(**kwargs):
                return RealChatOpenAI(**kwargs,http_async_client=client)
            req=server.Request(model=server.MODEL,messages=[server.Message(role='user',content='Where is Astranaar?')],response_format={'type':'json_object'})
            with patch.dict(os.environ,{'BERGET_API_KEY':'unit-test','BERGET_BASE_URL':'https://provider.invalid/v1'}),patch.object(server,'ChatOpenAI',factory):
                result=await server.completion(req)
            await client.aclose()
            self.assertEqual(result['usage'],{'prompt_tokens':100,'completion_tokens':20})
            self.assertEqual(result['choices'][0]['finish_reason'],'stop')
            self.assertIn('Astranaar',result['choices'][0]['message']['content'])
        asyncio.run(exercise())
        self.assertEqual(len(requests),1)
        self.assertEqual(os.environ['LANGSMITH_TRACING'],'false')

if __name__=='__main__': unittest.main()
