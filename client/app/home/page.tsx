import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  FileText,
  Zap,
  Users,
  Sparkles,
  Code,
  MessageSquare,
  ArrowRight,
  Check,
} from "lucide-react";

export default function page() {
  const [scrollY, setScrollY] = useState(0);
  const [activeFeature, setActiveFeature] = useState(0);

  useEffect(() => {
    const handleScroll = () => setScrollY(window.scrollY);
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  useEffect(() => {
    const interval = setInterval(() => {
      setActiveFeature((prev) => (prev + 1) % 3);
    }, 3000);
    return () => clearInterval(interval);
  }, []);

  const features = [
    {
      icon: <Users className="w-6 h-6" />,
      title: "Real-Time Collaboration",
      description:
        "Work together seamlessly with live cursors, instant updates, and presence awareness.",
    },
    {
      icon: <Sparkles className="w-6 h-6" />,
      title: "AI-Powered Assistance",
      description:
        "Intelligent copilots help you write, refactor, and document faster than ever.",
    },
    {
      icon: <Code className="w-6 h-6" />,
      title: "Built for Technical Teams",
      description:
        "Rich code blocks, syntax highlighting, and seamless integration with your workflow.",
    },
  ];

  const capabilities = [
    "Rich text editing with advanced formatting",
    "Live collaborative editing with CRDTs",
    "Code syntax highlighting",
    "Real-time presence indicators",
    "Version history tracking",
    "AI-powered suggestions",
  ];

  return (
    <div className="min-h-screen bg-white text-black overflow-hidden">
      {/* Animated background elements */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div
          className="absolute w-96 h-96 bg-black/5 rounded-full blur-3xl -top-48 -left-48 animate-pulse"
          style={{ animationDuration: "4s" }}
        />
        <div
          className="absolute w-96 h-96 bg-black/5 rounded-full blur-3xl top-1/2 -right-48 animate-pulse"
          style={{ animationDuration: "6s", animationDelay: "1s" }}
        />
        <div
          className="absolute w-96 h-96 bg-black/5 rounded-full blur-3xl -bottom-48 left-1/3 animate-pulse"
          style={{ animationDuration: "5s", animationDelay: "2s" }}
        />
      </div>

      {/* Navigation */}
      <nav className="relative z-10 flex items-center justify-between px-8 py-6 max-w-7xl mx-auto border-b border-black/10">
        <div className="flex items-center space-x-2">
          <div className="w-8 h-8 bg-black rounded-lg flex items-center justify-center">
            <FileText className="w-5 h-5 text-white" />
          </div>
          <span className="text-2xl font-bold">Kairo</span>
        </div>
        <div className="flex items-center space-x-4">
          <Button
            variant="ghost"
            className="text-black/70 hover:text-black hover:bg-black/5"
          >
            Features
          </Button>
          <Button
            variant="ghost"
            className="text-black/70 hover:text-black hover:bg-black/5"
          >
            Documentation
          </Button>
          <Button className="bg-black text-white hover:bg-black/90">
            Get Started
          </Button>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="relative z-10 max-w-7xl mx-auto px-8 pt-20 pb-32">
        <div className="text-center space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-1000">
          <div className="inline-flex items-center space-x-2 px-4 py-2 bg-black/5 rounded-full border border-black/10">
            <Zap className="w-4 h-4" />
            <span className="text-sm">Real-time collaboration platform</span>
          </div>

          <h1 className="text-6xl md:text-7xl font-bold leading-tight tracking-tight">
            Technical Documentation
            <br />
            <span className="italic">Reimagined</span>
          </h1>

          <p className="text-xl text-black/60 max-w-2xl mx-auto">
            The intelligent workspace where technical teams write, collaborate,
            and ship documentation at the speed of thought.
          </p>

          <div className="flex items-center justify-center space-x-4 pt-4">
            <Button
              size="lg"
              className="bg-black text-white hover:bg-black/90 text-lg px-8 group"
            >
              Start Creating
              <ArrowRight className="w-5 h-5 ml-2 group-hover:translate-x-1 transition-transform" />
            </Button>
            <Button
              size="lg"
              variant="outline"
              className="border-black/20 hover:bg-black/5 text-lg px-8"
            >
              View Demo
            </Button>
          </div>
        </div>

        {/* Floating Editor Preview */}
        <div
          className="mt-20 relative animate-in fade-in slide-in-from-bottom-8 duration-1000 delay-300"
          style={{ transform: `translateY(${scrollY * 0.1}px)` }}
        >
          <div className="absolute inset-0 bg-black/5 blur-3xl" />
          <Card className="relative bg-white border-black/20 shadow-2xl overflow-hidden">
            <CardContent className="p-0">
              <div className="bg-black/5 border-b border-black/10 px-6 py-3 flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="flex space-x-2">
                    <div className="w-3 h-3 rounded-full bg-black/20" />
                    <div className="w-3 h-3 rounded-full bg-black/20" />
                    <div className="w-3 h-3 rounded-full bg-black/20" />
                  </div>
                  <span className="text-sm text-black/60">
                    Project Documentation
                  </span>
                </div>
                <div className="flex items-center space-x-2">
                  <div className="flex -space-x-2">
                    {[1, 2, 3].map((i) => (
                      <div
                        key={i}
                        className="w-7 h-7 rounded-full bg-black text-white border-2 border-white flex items-center justify-center text-xs font-medium"
                      >
                        {String.fromCharCode(64 + i)}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
              <div className="p-8 space-y-4">
                {[1, 2, 3, 4].map((i) => (
                  <div
                    key={i}
                    className="space-y-2 animate-in fade-in slide-in-from-left duration-700"
                    style={{ animationDelay: `${i * 100}ms` }}
                  >
                    <div className="h-4 bg-black/5 rounded w-3/4" />
                    <div className="h-4 bg-black/5 rounded w-full" />
                    <div className="h-4 bg-black/5 rounded w-5/6" />
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* Features Section */}
      <section className="relative z-10 max-w-7xl mx-auto px-8 py-32 bg-black/[0.02]">
        <div className="text-center mb-16">
          <h2 className="text-4xl font-bold mb-4">Everything You Need</h2>
          <p className="text-black/60 text-lg">
            Powerful features designed for technical teams
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-8">
          {features.map((feature, index) => (
            <Card
              key={index}
              className={`bg-white border-black/10 hover:border-black/30 transition-all duration-300 hover:scale-105 hover:shadow-xl cursor-pointer ${
                activeFeature === index ? "border-black/30 shadow-xl" : ""
              }`}
            >
              <CardContent className="p-8 space-y-4">
                <div className="w-12 h-12 bg-black/5 rounded-lg flex items-center justify-center">
                  {feature.icon}
                </div>
                <h3 className="text-xl font-semibold">{feature.title}</h3>
                <p className="text-black/60">{feature.description}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Capabilities Section */}
      <section className="relative z-10 max-w-7xl mx-auto px-8 py-32">
        <div className="grid md:grid-cols-2 gap-16 items-center">
          <div className="space-y-6">
            <h2 className="text-4xl font-bold leading-tight">
              Built for the way
              <br />
              <span className="italic">technical teams work</span>
            </h2>
            <p className="text-black/60 text-lg">
              Every feature is designed to enhance your documentation workflow,
              from first draft to final review.
            </p>
            <div className="space-y-3">
              {capabilities.map((capability, index) => (
                <div
                  key={index}
                  className="flex items-center space-x-3 animate-in fade-in slide-in-from-left duration-500"
                  style={{ animationDelay: `${index * 100}ms` }}
                >
                  <div className="w-6 h-6 bg-black rounded-full flex items-center justify-center flex-shrink-0">
                    <Check className="w-4 h-4 text-white" />
                  </div>
                  <span className="text-black/80">{capability}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="relative">
            <div className="absolute inset-0 bg-black/5 blur-3xl" />
            <Card className="relative bg-white border-black/20 shadow-2xl">
              <CardContent className="p-8 space-y-6">
                <div className="flex items-center space-x-3">
                  <MessageSquare className="w-6 h-6" />
                  <span className="font-semibold">Live Collaboration</span>
                </div>
                <div className="space-y-4">
                  {[1, 2, 3].map((i) => (
                    <div
                      key={i}
                      className="flex items-start space-x-3 animate-in fade-in slide-in-from-bottom duration-700"
                      style={{ animationDelay: `${i * 200}ms` }}
                    >
                      <div className="w-8 h-8 rounded-full bg-black flex-shrink-0" />
                      <div className="flex-1 space-y-2">
                        <div className="h-3 bg-black/5 rounded w-3/4" />
                        <div className="h-3 bg-black/5 rounded w-1/2" />
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="relative z-10 max-w-7xl mx-auto px-8 py-32">
        <Card className="bg-black border-0 overflow-hidden shadow-2xl">
          <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGRlZnM+PHBhdHRlcm4gaWQ9ImdyaWQiIHdpZHRoPSI2MCIgaGVpZ2h0PSI2MCIgcGF0dGVyblVuaXRzPSJ1c2VyU3BhY2VPblVzZSI+PHBhdGggZD0iTSAxMCAwIEwgMCAwIDAgMTAiIGZpbGw9Im5vbmUiIHN0cm9rZT0id2hpdGUiIHN0cm9rZS1vcGFjaXR5PSIwLjEiIHN0cm9rZS13aWR0aD0iMSIvPjwvcGF0dGVybj48L2RlZnM+PHJlY3Qgd2lkdGg9IjEwMCUiIGhlaWdodD0iMTAwJSIgZmlsbD0idXJsKCNncmlkKSIvPjwvc3ZnPg==')] opacity-20" />
          <CardContent className="relative p-16 text-center space-y-6">
            <h2 className="text-4xl font-bold text-white">
              Ready to Transform Your Workflow?
            </h2>
            <p className="text-xl text-white/80 max-w-2xl mx-auto">
              Join technical teams who are already creating better
              documentation, faster.
            </p>
            <div className="flex items-center justify-center space-x-4 pt-4">
              <Button
                size="lg"
                className="bg-white text-black hover:bg-white/90 text-lg px-8"
              >
                Get Started Free
              </Button>
              <Button
                size="lg"
                variant="outline"
                className="border-white text-white hover:bg-white/10 text-lg px-8"
              >
                Schedule Demo
              </Button>
            </div>
          </CardContent>
        </Card>
      </section>

      {/* Footer */}
      <footer className="relative z-10 border-t border-black/10 mt-32">
        <div className="max-w-7xl mx-auto px-8 py-12">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-black rounded-lg flex items-center justify-center">
                <FileText className="w-5 h-5 text-white" />
              </div>
              <span className="text-xl font-bold">Kairo</span>
            </div>
            <p className="text-black/40 text-sm">
              © 2025 Kairo. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </div>
  );
}
