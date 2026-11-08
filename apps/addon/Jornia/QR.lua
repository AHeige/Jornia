-- First-party QR byte encoder: ISO/IEC 18004 Model 2, versions 4-L and 16-L.
-- Fixed mask 0, ECI 26 (UTF-8), Reed-Solomon blocks and standard quiet zone.
-- This format primitive owns no chat, identity, consent or retry decisions.
local QR={}
local function xor(a,b)
 local n,p=0,1
 while a>0 or b>0 do if a%2~=b%2 then n=n+p end;a=math.floor(a/2);b=math.floor(b/2);p=p*2 end
 return n
end
local exp,log={},{};local x=1
for i=0,254 do exp[i]=x;log[x]=i;x=x*2;if x>=256 then x=xor(x,285)end end
for i=255,509 do exp[i]=exp[i-255]end
local function mul(a,b)if a==0 or b==0 then return 0 end;return exp[log[a]+log[b]]end
local generators={}
local function parity(data,count)
 local g=generators[count]
 if not g then
  g={1}
  for i=0,count-1 do local n={};for j=1,#g+1 do n[j]=0 end;for j,c in ipairs(g)do n[j]=xor(n[j],c);n[j+1]=xor(n[j+1],mul(c,exp[i]))end;g=n end
  generators[count]=g
 end
 local rem={};for i=1,count do rem[i]=0 end
 for _,v in ipairs(data)do local factor=xor(v,rem[1]);table.remove(rem,1);rem[count]=0;for j=1,count do rem[j]=xor(rem[j],mul(g[j+1],factor))end end
 return rem
end
function QR.Encode(text)
 assert(type(text)=='string' and #text<=584,'QR payload too large')
 local version=#text<=76 and 4 or 16
 local size=version*4+17;local capacity=version==4 and 80 or 589
 local bits={}
 local function append(value,n)for i=n-1,0,-1 do bits[#bits+1]=math.floor(value/2^i)%2 end end
 append(7,4);append(26,8);append(4,4);append(#text,version==4 and 8 or 16)
 for i=1,#text do append(text:byte(i),8)end
 assert(#bits<=capacity*8,'QR payload too large')
 append(0,math.min(4,capacity*8-#bits));while #bits%8~=0 do append(0,1)end
 local data={};for i=1,#bits,8 do local v=0;for j=i,i+7 do v=v*2+bits[j]end;data[#data+1]=v end
 local pad=236;while #data<capacity do data[#data+1]=pad;pad=pad==236 and 17 or 236 end
 local blocks,ecc={},{};local at=1;local count=version==4 and 1 or 6;local ec=version==4 and 20 or 24
 for b=1,count do local n=version==4 and 80 or (b==6 and 99 or 98);local block={};for j=1,n do block[j]=data[at];at=at+1 end;blocks[b]=block;ecc[b]=parity(block,ec)end
 local wire={};for j=1,(version==4 and 80 or 99)do for b=1,count do if blocks[b][j]then wire[#wire+1]=blocks[b][j]end end end
 for j=1,ec do for b=1,count do wire[#wire+1]=ecc[b][j]end end
 local matrix,reserved={},{};for y=0,size-1 do matrix[y]={};reserved[y]={}end
 local function set(px,py,dark)if px>=0 and py>=0 and px<size and py<size then matrix[py][px]=dark;reserved[py][px]=true end end
 for i=0,size-1 do set(6,i,i%2==0);set(i,6,i%2==0)end
 for _,center in ipairs({{3,3},{size-4,3},{3,size-4}})do for dy=-4,4 do for dx=-4,4 do local d=math.max(math.abs(dx),math.abs(dy));set(center[1]+dx,center[2]+dy,d~=2 and d~=4)end end end
 local centers=version==4 and {6,26} or {6,26,50,74}
 for iy,cy in ipairs(centers)do for ix,cx in ipairs(centers)do
  if not((ix==1 and iy==1)or(ix==1 and iy==#centers)or(ix==#centers and iy==1))then for dy=-2,2 do for dx=-2,2 do set(cx+dx,cy+dy,math.max(math.abs(dx),math.abs(dy))~=1)end end end
 end end
 -- L correction is format value 01; mask 000. BCH(15,5), polynomial 0x537.
 local format=8*1024;for i=14,10,-1 do if math.floor(format/2^i)%2==1 then format=xor(format,1335*2^(i-10))end end
 format=xor(xor(8*1024,format),21522)
 local function f(i)return math.floor(format/2^i)%2==1 end
 for i=0,5 do set(8,i,f(i))end;set(8,7,f(6));set(8,8,f(7));set(7,8,f(8))
 for i=9,14 do set(14-i,8,f(i))end
 for i=0,7 do set(size-1-i,8,f(i))end;for i=8,14 do set(8,size-15+i,f(i))end;set(8,size-8,true)
 if version>=7 then
  local rem=version*4096;for i=17,12,-1 do if math.floor(rem/2^i)%2==1 then rem=xor(rem,7973*2^(i-12))end end
  local code=version*4096+rem
  for i=0,17 do local a=size-11+i%3;local b=math.floor(i/3);local v=math.floor(code/2^i)%2==1;set(a,b,v);set(b,a,v)end
 end
 local index,right,up=0,size-1,true
 while right>=1 do
  if right==6 then right=5 end
  for row=0,size-1 do local y=up and size-1-row or row
   for dx=0,1 do local col=right-dx
    if not reserved[y][col]then
     local byte=wire[math.floor(index/8)+1]or 0;local v=math.floor(byte/2^(7-index%8))%2==1;index=index+1
     if (col+y)%2==0 then v=not v end;matrix[y][col]=v
    end
   end
  end
  right=right-2;up=not up
 end
 return matrix,size
end
JorniaQR=QR
