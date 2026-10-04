# Built from the repository root (build.context in service.yaml) so the image
# can carry the documentation it answers questions about.
FROM python:3.14-slim AS build
RUN python -m venv /opt/venv
COPY apps/rag-assistant/requirements.txt .
RUN /opt/venv/bin/pip install --no-cache-dir -r requirements.txt
# Bake the embedding model in: no download at startup, works offline.
ARG EMBEDDING_MODEL=sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2
RUN /opt/venv/bin/python -c "from fastembed import TextEmbedding; TextEmbedding('${EMBEDDING_MODEL}', cache_dir='/opt/models')" \
    && chmod -R a+rX /opt/models
# pip is not needed at runtime (and vendors its own dependencies).
RUN /opt/venv/bin/python -m pip uninstall -y -q pip

FROM python:3.14-slim
# Security fixes published since the base image was built; no pip at runtime.
RUN apt-get update && apt-get upgrade -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/* \
    && python -m pip uninstall -y -q pip
ENV PATH=/opt/venv/bin:$PATH \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    HF_HUB_OFFLINE=1 \
    EMBEDDING_CACHE_DIR=/opt/models \
    CORPUS_DIR=/srv/corpus
COPY --from=build /opt/venv /opt/venv
COPY --from=build /opt/models /opt/models
WORKDIR /srv
COPY apps/rag-assistant/app ./app
# The corpus: platform docs, ADRs and both READMEs.
COPY docs ./corpus/docs
COPY README.md README.en.md ./corpus/
# Numeric UID so Kubernetes can verify runAsNonRoot.
USER 10001
EXPOSE 8080
CMD ["python", "-m", "app"]
