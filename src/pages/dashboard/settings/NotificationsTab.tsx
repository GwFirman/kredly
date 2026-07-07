import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';

export default function NotificationsTab() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Notifikasi</h2>
        <p className="text-sm text-muted-foreground">
          Atur preferensi notifikasi Kamu
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Notifikasi Email</CardTitle>
          <CardDescription>
            Pilih notifikasi yang ingin Anda terima via email
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Kredensial Baru</p>
              <p className="text-sm text-muted-foreground">
                Notifikasi saat kredensial berhasil diterbitkan
              </p>
            </div>
            <Button variant="outline" size="sm">
              Aktif
            </Button>
          </div>
          <Separator />
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Update Asesmen</p>
              <p className="text-sm text-muted-foreground">
                Notifikasi saat asesmen selesai diproses
              </p>
            </div>
            <Button variant="outline" size="sm">
              Aktif
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
