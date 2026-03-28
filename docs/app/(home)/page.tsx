import Link from 'next/link';

export default function HomePage() {
  return (
    <div className="flex flex-col justify-center text-center flex-1 gap-4 py-16">
      <h1 className="text-4xl font-bold">Mannaiah Docs</h1>
      <p className="text-fd-muted-foreground max-w-lg mx-auto">
        The internal knowledge base for Mannaiah — the all-in-one business
        administration backend powering{' '}
        <a
          href="https://www.flockstore.co/"
          className="font-medium underline"
          target="_blank"
          rel="noopener noreferrer"
        >
          Flock
        </a>
        .
      </p>
      <div>
        <Link
          href="/docs"
          className="rounded-full bg-fd-primary text-fd-primary-foreground px-5 py-2 text-sm font-medium"
        >
          Browse the docs →
        </Link>
      </div>
    </div>
  );
}
