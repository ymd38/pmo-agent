<script setup lang="ts">
// 公開LP（未ログインでも閲覧可）。認証ミドルウェアは付けない。
const capabilities = [
  'プロジェクトコード発行',
  '工数・コスト集計',
  'Backlog 連携',
  'AI 経営レポート',
]

const features = [
  {
    title: '発行後は不変のプロジェクトコード',
    body: 'プログラムでプレフィックスを発行し、承認時に枝番を採番。全社のプロジェクトを横断する一意の業務キーになる。',
  },
  {
    title: '内部工数を含む“真のコスト”',
    body: '外注費に、グレード別単価で算出した内部工数コストを加算。プロジェクトの実コストを初めて可視化する。',
  },
  {
    title: '権限はロールで厳密に制御',
    body: '単価やコスト内訳は見える人にだけ。スコープ制御はAPIミドルウェアで強制し、表示制御に依存しない。',
  },
  {
    title: 'レポートは自動で生成',
    body: 'Backlog の進捗を日次収集し、AIが進捗・リスク・トレンドを要約。日次・週次・経営レポートへ。',
  },
]

const steps = [
  { no: '01', title: '起案・コード発行', body: 'プログラム配下にプロジェクトを起案。PMO管理者が審査し、コードを発行して進行中へ。' },
  { no: '02', title: '工数とコストを集計', body: 'メンバーが日次で工数を入力。外注費と内部工数から真のコストを算出する。' },
  { no: '03', title: '経営の言葉で報告', body: 'AIが横断的に総括し、経営層向けエグゼクティブレポートを毎週自動生成する。' },
]
</script>

<template>
  <div class="min-h-screen bg-canvas text-ink">
    <LandingNav />

    <main>
      <!-- ============================ HERO ============================ -->
      <section class="relative overflow-hidden">
        <div class="hero-aura pointer-events-none absolute inset-0" aria-hidden="true" />

        <div class="relative mx-auto max-w-6xl px-6 pt-24 pb-20 md:pt-36 md:pb-28">
          <p class="eyebrow text-ink-muted">Project Governance Platform</p>

          <h1 class="display-xxl mt-6 text-ink">
            プロジェクトを、<br>統べる。
          </h1>

          <p class="subhead mt-8 max-w-2xl text-ink-muted">
            コード発行から工数・コスト、経営レポートまで。<br class="hidden sm:block">
            組織のすべてのプロジェクトを、ひとつの統制基盤に。
          </p>

          <div class="mt-10 flex flex-wrap items-center gap-3">
            <PillButton variant="primary" to="/login">ログイン</PillButton>
            <PillButton variant="secondary" href="#capabilities">できることを見る</PillButton>
          </div>

          <ul class="mt-14 flex flex-wrap items-center gap-x-6 gap-y-2">
            <li
              v-for="(cap, i) in capabilities"
              :key="cap"
              class="flex items-center gap-6 text-sm text-ink-muted"
            >
              <span
                v-if="i !== 0"
                class="hidden size-1 rounded-full bg-ink-muted/50 sm:block"
                aria-hidden="true"
              />
              {{ cap }}
            </li>
          </ul>
        </div>
      </section>

      <!-- ======================= CAPABILITIES ========================= -->
      <section id="capabilities" class="mx-auto max-w-6xl px-6 py-24 md:py-32">
        <p class="eyebrow text-ink-muted">Capabilities</p>
        <h2 class="display-lg mt-5 max-w-3xl text-ink">
          断片化した管理を、ひとつに。
        </h2>
        <p class="body-lg mt-5 max-w-2xl text-ink-muted">
          Excel とベンダーツールに散らばった情報を、統制の効く一枚の基盤へ集約する。
        </p>

        <div class="mt-14 grid gap-5 md:grid-cols-3">
          <GradientSpotlight
            variant="violet"
            eyebrow="Cost"
            title="工数とコストの可視化"
          >
            外注費に内部工数コストを重ね、プロジェクトの真のコストを一目で。
          </GradientSpotlight>

          <GradientSpotlight
            variant="magenta"
            eyebrow="Insight"
            title="AIが進捗とリスクを要約"
          >
            Backlog の進捗をAIが日次で分析し、リスクとトレンドを言語化する。
          </GradientSpotlight>

          <GradientSpotlight
            variant="orange"
            eyebrow="Control"
            title="コードで全社を統制"
          >
            発行後は不変の業務キーで、すべてのプロジェクトを横断管理する。
          </GradientSpotlight>
        </div>
      </section>

      <!-- ========================= FEATURES =========================== -->
      <section class="border-t border-hairline-soft">
        <div class="mx-auto max-w-6xl px-6 py-24 md:py-32">
          <h2 class="display-lg max-w-3xl text-ink">統制に必要な、すべて。</h2>

          <div class="mt-14 grid gap-x-12 gap-y-12 md:grid-cols-2">
            <div
              v-for="feature in features"
              :key="feature.title"
              class="border-t border-hairline pt-6"
            >
              <h3 class="display-md text-ink">{{ feature.title }}</h3>
              <p class="body-lg mt-3 max-w-md text-ink-muted">{{ feature.body }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- ========================== HOW =============================== -->
      <section id="how" class="border-t border-hairline-soft">
        <div class="mx-auto max-w-6xl px-6 py-24 md:py-32">
          <p class="eyebrow text-ink-muted">How it works</p>
          <h2 class="display-lg mt-5 max-w-3xl text-ink">起案から、経営報告まで。</h2>

          <div class="mt-14 grid gap-5 md:grid-cols-3">
            <div
              v-for="step in steps"
              :key="step.no"
              class="rounded-xl bg-surface-1 p-8"
            >
              <span class="display-md text-ink-muted">{{ step.no }}</span>
              <h3 class="mt-6 text-lg font-semibold tracking-tight text-ink">{{ step.title }}</h3>
              <p class="mt-3 text-sm leading-relaxed text-ink-muted">{{ step.body }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- =========================== CTA ============================== -->
      <section id="roadmap" class="border-t border-hairline-soft">
        <div class="relative overflow-hidden">
          <div class="hero-aura pointer-events-none absolute inset-0" aria-hidden="true" />
          <div class="relative mx-auto max-w-6xl px-6 py-28 text-center md:py-36">
            <h2 class="display-xl mx-auto max-w-4xl text-ink">
              プロジェクトの今を、<br class="hidden sm:block">経営の言葉に。
            </h2>
            <p class="body-lg mx-auto mt-6 max-w-xl text-ink-muted">
              いまは事前開発フェーズ。基盤はここから育てていく。
            </p>
            <div class="mt-10 flex justify-center">
              <PillButton variant="primary" to="/login">ログイン</PillButton>
            </div>
          </div>
        </div>
      </section>
    </main>

    <SiteFooter />
  </div>
</template>
